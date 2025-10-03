package python

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// filterVSCodeOutput removes VS Code specific output that can interfere with results
func filterVSCodeOutput(output string) string {
	// Remove VS Code Native REPL text that sometimes appears
	filtered := strings.ReplaceAll(output, "Cmd click to launch VS Code Native REPL", "")
	// Remove any other VS Code related text that might appear
	filtered = strings.ReplaceAll(filtered, "VS Code Native REPL", "")
	return filtered
}

// readOutput reads from a pipe (stdout) and sends buffered output to a channel
func (pr *PythonRuntime) readOutput(pipe io.ReadCloser, ch chan<- string) {
	scanner := bufio.NewScanner(pipe)
	var outputBuffer strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		// Python's interactive mode prints prompts we need to ignore
		if strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "...") {
			continue
		}
		if strings.HasPrefix(line, EndOfOutputMarker) {
			// Extract the execution ID from the marker
			markerParts := strings.Split(line, "-")
			if len(markerParts) >= 2 {
				// This is a unique marker with execution ID
				// Process buffered output: separate print() output from JSON result
				bufferedOutput := outputBuffer.String()
				// Filter out VS Code specific output
				bufferedOutput = filterVSCodeOutput(bufferedOutput)
				if pr.verbose {
					fmt.Printf("DEBUG: Python output buffer before processing: '%s'\n", bufferedOutput)
				}
				lines := strings.Split(strings.TrimSpace(bufferedOutput), "\n")
				if pr.verbose {
					fmt.Printf("DEBUG: Python output split into %d lines: %v\n", len(lines), lines)
				}

				if len(lines) > 0 {
					// Don't print to console here - let the engine handle output display
					// Print all lines except the last one (which is JSON result)
					// for i, line := range lines {
					//	if i < len(lines)-1 && line != "" {
					//		fmt.Println(line)
					//	}
					// }
					// Return only the last line (JSON result)
					resultLine := lines[len(lines)-1]
					if pr.verbose {
						fmt.Printf("DEBUG: Python returning result line: '%s'\n", resultLine)
					}
					ch <- resultLine
				} else {
					if pr.verbose {
						fmt.Printf("DEBUG: Python returning empty result (no lines)\n")
					}
					ch <- ""
				}
				outputBuffer.Reset()
			} else {
				// This is the old marker format, handle it for backward compatibility
				// Process buffered output: separate print() output from JSON result
				bufferedOutput := outputBuffer.String()
				// Filter out VS Code specific output
				bufferedOutput = filterVSCodeOutput(bufferedOutput)
				if pr.verbose {
					fmt.Printf("DEBUG: Python output buffer before processing (old format): '%s'\n", bufferedOutput)
				}
				lines := strings.Split(strings.TrimSpace(bufferedOutput), "\n")
				if pr.verbose {
					fmt.Printf("DEBUG: Python output split into %d lines (old format): %v\n", len(lines), lines)
				}

				if len(lines) > 0 {
					// Don't print to console here - let the engine handle output display
					// Print all lines except the last one (which is JSON result)
					// for i, line := range lines {
					//	if i < len(lines)-1 && line != "" {
					//		fmt.Println(line)
					//	}
					// }
					// Return only the last line (JSON result)
					resultLine := lines[len(lines)-1]
					if pr.verbose {
						fmt.Printf("DEBUG: Python returning result line (old format): '%s'\n", resultLine)
					}
					ch <- resultLine
				} else {
					if pr.verbose {
						fmt.Printf("DEBUG: Python returning empty result (no lines, old format)\n")
					}
					ch <- ""
				}
				outputBuffer.Reset()
			}
		} else {
			// Filter VS Code output before processing
			filteredLine := filterVSCodeOutput(line)
			// Only process non-empty lines after filtering
			if filteredLine != "" {
				outputBuffer.WriteString(filteredLine + "\n")
				// Also capture to outputCapture if it's set (for print function output)
				// Use mutex to safely access outputCapture
				pr.mutex.RLock()
				if pr.outputCapture != nil {
					if pr.verbose {
						fmt.Printf("DEBUG: readOutput - writing to outputCapture: '%s'\n", filteredLine)
					}
					pr.outputCapture.WriteString(filteredLine + "\n")
				} else {
					if pr.verbose {
						fmt.Printf("DEBUG: readOutput - outputCapture is nil, not writing: '%s'\n", filteredLine)
					}
				}
				pr.mutex.RUnlock()
			} else if pr.verbose && line != filteredLine {
				fmt.Printf("DEBUG: readOutput - filtered out VS Code text: '%s'\n", line)
			}
		}
	}
}

// readError reads from a pipe (stderr) and sends errors to a channel.
// It reads in small chunks to be responsive to errors as they appear on the stream.
func (pr *PythonRuntime) readError(pipe io.ReadCloser, ch chan<- error) {
	buf := make([]byte, 2048)
	for {
		n, err := pipe.Read(buf)
		if err != nil {
			// Pipe closed, exit goroutine
			if err == io.EOF {
				return
			}
			ch <- err
			return
		}
		if n > 0 {
			ch <- fmt.Errorf("python stderr: %s", string(buf[:n]))
		}
	}
}

// GetCapturedOutput returns the captured stdout output and clears the capture buffer
func (pr *PythonRuntime) GetCapturedOutput() string {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if pr.outputCapture == nil {
		if pr.verbose {
			fmt.Printf("DEBUG: GetCapturedOutput - outputCapture is nil\n")
		}
		return ""
	}

	captured := pr.outputCapture.String()
	if pr.verbose {
		fmt.Printf("DEBUG: GetCapturedOutput - captured: '%s'\n", captured)
	}

	// Strip the JSON result from captured output
	// The Python runtime outputs both print statements and JSON results,
	// but we only want the print output for GetCapturedOutput()

	// If the captured output ends with "null", strip it
	if strings.HasSuffix(captured, "null") {
		captured = strings.TrimSuffix(captured, "null")
		// Also remove potential newline before the null
		captured = strings.TrimSuffix(captured, "\n")
	}

	// Trim trailing newlines to avoid extra line breaks in output
	captured = strings.TrimSpace(captured)

	// Clear the outputCapture after reading it (like other runtimes do)
	pr.outputCapture.Reset()

	return captured
}

// ClearCapturedOutput clears the captured output buffer
func (pr *PythonRuntime) ClearCapturedOutput() {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if pr.outputCapture != nil {
		pr.outputCapture.Reset()
		if pr.verbose {
			fmt.Printf("DEBUG: ClearCapturedOutput - buffer cleared\n")
		}
	}
}
