package acceptancetests

import (
	"strings"
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestBitstringNestedConstruction tests creating nested bitstrings
func TestBitstringNestedConstruction(t *testing.T) {
	// Create inner bitstrings
	inner1, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(8)).
		AddInteger(2, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner1: %v", err)
	}
	if inner1 == nil {
		t.Fatal("inner1 is nil")
	}
	if inner1.Length() != 16 {
		t.Errorf("Expected inner1 length 16, got %d", inner1.Length())
	}

	inner2, err := builder.NewBuilder().
		AddInteger(3, bitstringpkg.WithSize(8)).
		AddInteger(4, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner2: %v", err)
	}
	if inner2 == nil {
		t.Fatal("inner2 is nil")
	}
	if inner2.Length() != 16 {
		t.Errorf("Expected inner2 length 16, got %d", inner2.Length())
	}

	// Create outer bitstring with nested ones
	outer, err := builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(inner1).
		AddBitstring(inner2).
		AddInteger(5, bitstringpkg.WithSize(8)).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if outer == nil {
		t.Fatal("outer is nil")
	}
	if outer.Length() != 48 { // 8 + 16 + 16 + 8 = 48 bits
		t.Errorf("Expected outer length 48, got %d", outer.Length())
	}
}

// TestBitstringNestedMatching tests matching nested bitstrings
func TestBitstringNestedMatching(t *testing.T) {
	// Create nested bitstring using AddBitstring functionality
	innerData, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(8)).
		AddInteger(2, bitstringpkg.WithSize(8)).
		AddInteger(3, bitstringpkg.WithSize(8)).
		AddInteger(4, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner bitstring: %v", err)
	}

	// Create outer bitstring with nested inner bitstring
	bs, err := builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(innerData).
		AddInteger(5, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create outer bitstring: %v", err)
	}

	var prefix int
	var dataSection *bitstringpkg.BitString
	var suffix int

	// Try to match nested structure
	results, err := matcher.NewMatcher().
		Integer(&prefix, bitstringpkg.WithSize(8)).
		Bitstring(&dataSection, bitstringpkg.WithSize(32)). // 4 bytes = 32 bits
		Integer(&suffix, bitstringpkg.WithSize(8)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if results == nil {
		t.Fatal("results is nil")
	}
	if prefix != 0 {
		t.Errorf("Expected prefix 0, got %d", prefix)
	}
	if suffix != 5 {
		t.Errorf("Expected suffix 5, got %d", suffix)
	}
	if dataSection == nil {
		t.Fatal("dataSection is nil")
	}
	if dataSection.Length() != 32 {
		t.Errorf("Expected dataSection length 32, got %d", dataSection.Length())
	}

	// Now check the content of the nested bitstring
	var a, b, c, d int
	nestedResults, err := matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Integer(&d, bitstringpkg.WithSize(8)).
		Match(dataSection)

	if err != nil {
		t.Fatalf("Expected no error for nested matching, got %v", err)
	}
	if nestedResults == nil {
		t.Fatal("nestedResults is nil")
	}
	if a != 1 || b != 2 || c != 3 || d != 4 {
		t.Errorf("Expected a=1, b=2, c=3, d=4, got a=%d, b=%d, c=%d, d=%d", a, b, c, d)
	}
}

// TestBitstringComplexNestedStructure tests complex nested structure
func TestBitstringComplexNestedStructure(t *testing.T) {
	// This test should fail because complex nesting is not yet implemented

	// Create multiple levels of nesting
	level1, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(4)).
		AddInteger(2, bitstringpkg.WithSize(4)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create level1: %v", err)
	}

	level2, err := builder.NewBuilder().
		AddBitstring(level1).
		AddInteger(3, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create level2: %v", err)
	}

	level3, err := builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(level2).
		AddInteger(4, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create level3: %v", err)
	}

	if level3 == nil {
		t.Fatal("level3 is nil")
	}
	if level3.Length() != 32 { // 8 + 16 + 8 = 32 bits
		t.Errorf("Expected level3 length 32, got %d", level3.Length())
	}

	// Check matching of complex structure
	var prefix int
	var middle *bitstringpkg.BitString
	var suffix int

	_, err = matcher.NewMatcher().
		Integer(&prefix, bitstringpkg.WithSize(8)).
		Bitstring(&middle, bitstringpkg.WithSize(16)).
		Integer(&suffix, bitstringpkg.WithSize(8)).
		Match(level3)

	if err != nil {
		t.Fatalf("Expected no error for complex nested matching, got %v", err)
	}
	if prefix != 0 {
		t.Errorf("Expected prefix 0, got %d", prefix)
	}
	if suffix != 4 {
		t.Errorf("Expected suffix 4, got %d", suffix)
	}
	if middle == nil {
		t.Fatal("middle is nil")
	}
	if middle.Length() != 16 {
		t.Errorf("Expected middle length 16, got %d", middle.Length())
	}
}

// TestBitstringNestedErrorHandling tests error handling when working with nested structures
func TestBitstringNestedErrorHandling(t *testing.T) {
	// Create bitstring
	inner, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(8)).
		AddInteger(2, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner: %v", err)
	}

	// Try to create bitstring with wrong size
	_, err = builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(inner, bitstringpkg.WithSize(17)). // Odd size for bitstring
		AddInteger(3, bitstringpkg.WithSize(8)).
		Build()

	// Should be an error due to size mismatch
	if err == nil {
		t.Fatal("Expected error for size mismatch, got nil")
	}
	if err.Error() == "" {
		t.Fatal("Expected error message, got empty string")
	}
	// The error message now contains more specific information about bitstring size mismatch
	if !strings.Contains(err.Error(), "size mismatch") && !strings.Contains(err.Error(), "insufficient") && !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "less than") {
		t.Errorf("Expected error message to contain 'size mismatch', 'insufficient', 'invalid', or 'less than', got: %s", err.Error())
	}
}

// TestBitstringComplexNestedStructures tests complex nested structures
func TestBitstringComplexNestedStructures(t *testing.T) {
	// Test 1: Three-level nested structure
	innerMost, err := builder.NewBuilder().
		AddInteger(255, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create innerMost: %v", err)
	}
	if innerMost.Length() != 8 {
		t.Errorf("Expected innerMost length 8, got %d", innerMost.Length())
	}

	middle, err := builder.NewBuilder().
		AddInteger(128, bitstringpkg.WithSize(8)).
		AddBitstring(innerMost).
		AddInteger(64, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create middle: %v", err)
	}
	if middle.Length() != 24 {
		t.Errorf("Expected middle length 24, got %d", middle.Length())
	}

	outer, err := builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(middle).
		AddInteger(1, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create outer: %v", err)
	}
	if outer.Length() != 40 {
		t.Errorf("Expected outer length 40, got %d", outer.Length())
	}

	// Test matching the three-level structure
	var prefix, suffix int
	var matchedMiddle *bitstringpkg.BitString
	results, err := matcher.NewMatcher().
		Integer(&prefix, bitstringpkg.WithSize(8)).
		Bitstring(&matchedMiddle, bitstringpkg.WithSize(24)).
		Integer(&suffix, bitstringpkg.WithSize(8)).
		Match(outer)
	if err != nil {
		t.Fatalf("Failed to match outer: %v", err)
	}
	if !results[1].Matched {
		t.Fatal("Expected middle segment to match")
	}
	if matchedMiddle == nil {
		t.Fatal("matchedMiddle is nil")
	}
	if matchedMiddle.Length() != 24 {
		t.Errorf("Expected matchedMiddle length 24, got %d", matchedMiddle.Length())
	}
	if prefix != 0 || suffix != 1 {
		t.Errorf("Expected prefix=0, suffix=1, got prefix=%d, suffix=%d", prefix, suffix)
	}

	// Test matching the middle structure
	var middlePrefix, middleSuffix int
	var matchedInnerMost *bitstringpkg.BitString
	results2, err := matcher.NewMatcher().
		Integer(&middlePrefix, bitstringpkg.WithSize(8)).
		Bitstring(&matchedInnerMost, bitstringpkg.WithSize(8)).
		Integer(&middleSuffix, bitstringpkg.WithSize(8)).
		Match(matchedMiddle)
	if err != nil {
		t.Fatalf("Failed to match middle: %v", err)
	}
	if !results2[1].Matched {
		t.Fatal("Expected innerMost segment to match")
	}
	if matchedInnerMost == nil {
		t.Fatal("matchedInnerMost is nil")
	}
	if matchedInnerMost.Length() != 8 {
		t.Errorf("Expected matchedInnerMost length 8, got %d", matchedInnerMost.Length())
	}
	if middlePrefix != 128 || middleSuffix != 64 {
		t.Errorf("Expected middlePrefix=128, middleSuffix=64, got middlePrefix=%d, middleSuffix=%d", middlePrefix, middleSuffix)
	}

	// Test 2: Multiple nested bitstrings at the same level
	inner1, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(4)).
		AddInteger(2, bitstringpkg.WithSize(4)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner1: %v", err)
	}

	inner2, err := builder.NewBuilder().
		AddInteger(3, bitstringpkg.WithSize(8)).
		AddInteger(4, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner2: %v", err)
	}

	inner3, err := builder.NewBuilder().
		AddInteger(5, bitstringpkg.WithSize(16)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create inner3: %v", err)
	}

	multiNested, err := builder.NewBuilder().
		AddInteger(0xAA, bitstringpkg.WithSize(8)).
		AddBitstring(inner1).
		AddBitstring(inner2).
		AddBitstring(inner3).
		AddInteger(0xBB, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create multiNested: %v", err)
	}
	// inner1: 8 bits, inner2: 16 bits, inner3: 16 bits
	// Total: 8 + 8 + 16 + 16 + 8 = 56 bits
	if multiNested.Length() != 56 {
		t.Errorf("Expected multiNested length 56, got %d", multiNested.Length())
	}

	// Test matching multiple nested bitstrings
	var multiPrefix, multiSuffix int
	var matchedInner1, matchedInner2, matchedInner3 *bitstringpkg.BitString
	results3, err := matcher.NewMatcher().
		Integer(&multiPrefix, bitstringpkg.WithSize(8)).
		Bitstring(&matchedInner1, bitstringpkg.WithSize(8)).
		Bitstring(&matchedInner2, bitstringpkg.WithSize(16)).
		Bitstring(&matchedInner3, bitstringpkg.WithSize(16)).
		Integer(&multiSuffix, bitstringpkg.WithSize(8)).
		Match(multiNested)
	if err != nil {
		t.Fatalf("Failed to match multiNested: %v", err)
	}
	if !results3[1].Matched || !results3[2].Matched || !results3[3].Matched {
		t.Fatal("Expected all inner segments to match")
	}
	if matchedInner1.Length() != 8 {
		t.Errorf("Expected matchedInner1 length 8, got %d", matchedInner1.Length())
	}
	if matchedInner2.Length() != 16 {
		t.Errorf("Expected matchedInner2 length 16, got %d", matchedInner2.Length())
	}
	if matchedInner3.Length() != 16 {
		t.Errorf("Expected matchedInner3 length 16, got %d", matchedInner3.Length())
	}
	if multiPrefix != 0xAA || multiSuffix != 0xBB {
		t.Errorf("Expected multiPrefix=0xAA, multiSuffix=0xBB, got multiPrefix=%d, multiSuffix=%d", multiPrefix, multiSuffix)
	}

	// Test 3: Nested bitstring with mixed types
	mixedInner, err := builder.NewBuilder().
		AddInteger(42, bitstringpkg.WithSize(8)).
		AddBinary([]byte{0xDE, 0xAD, 0xBE, 0xEF}).
		AddInteger(99, bitstringpkg.WithSize(16)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create mixedInner: %v", err)
	}

	mixedOuter, err := builder.NewBuilder().
		AddInteger(0x55, bitstringpkg.WithSize(8)).
		AddBitstring(mixedInner).
		AddBinary([]byte{0xCA, 0xFE}).
		Build()
	if err != nil {
		t.Fatalf("Failed to create mixedOuter: %v", err)
	}
	// mixedInner: 8 + 32 + 16 = 56 bits
	// Total: 8 + 56 + 16 = 80 bits
	if mixedOuter.Length() != 80 {
		t.Errorf("Expected mixedOuter length 80, got %d", mixedOuter.Length())
	}

	// Test matching mixed nested structure
	var mixedPrefix int
	var matchedMixedInner *bitstringpkg.BitString
	var matchedBinary []byte
	results4, err := matcher.NewMatcher().
		Integer(&mixedPrefix, bitstringpkg.WithSize(8)).
		Bitstring(&matchedMixedInner, bitstringpkg.WithSize(56)).
		Binary(&matchedBinary, bitstringpkg.WithSize(2), bitstringpkg.WithUnit(8)). // 2 bytes * 8 = 16 bits
		Match(mixedOuter)
	if err != nil {
		t.Fatalf("Failed to match mixedOuter: %v", err)
	}
	if !results4[1].Matched || !results4[2].Matched {
		t.Fatal("Expected mixed inner and binary segments to match")
	}
	if matchedMixedInner == nil {
		t.Fatal("matchedMixedInner is nil")
	}
	if matchedMixedInner.Length() != 56 {
		t.Errorf("Expected matchedMixedInner length 56, got %d", matchedMixedInner.Length())
	}
	if mixedPrefix != 0x55 {
		t.Errorf("Expected mixedPrefix=0x55, got %d", mixedPrefix)
	}
	if string(matchedBinary) != string([]byte{0xCA, 0xFE}) {
		t.Errorf("Expected matchedBinary %v, got %v", []byte{0xCA, 0xFE}, matchedBinary)
	}
}

// TestBitstringNestedEdgeCases tests edge cases of nested structures
func TestBitstringNestedEdgeCases(t *testing.T) {
	// Test 1: Minimal nested bitstring (1 bit instead of empty)
	minimalInner, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(1)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create minimalInner: %v", err)
	}

	outerWithMinimal, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(8)).
		AddBitstring(minimalInner).
		AddInteger(2, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create outerWithMinimal: %v", err)
	}
	if outerWithMinimal.Length() != 17 { // 8 + 1 + 8 = 17 bits
		t.Errorf("Expected outerWithMinimal length 17, got %d", outerWithMinimal.Length())
	}

	// Test matching minimal nested bitstring
	var minimalPrefix, minimalSuffix int
	var matchedMinimal *bitstringpkg.BitString
	results, err := matcher.NewMatcher().
		Integer(&minimalPrefix, bitstringpkg.WithSize(8)).
		Bitstring(&matchedMinimal, bitstringpkg.WithSize(1)).
		Integer(&minimalSuffix, bitstringpkg.WithSize(8)).
		Match(outerWithMinimal)
	if err != nil {
		t.Fatalf("Failed to match outerWithMinimal: %v", err)
	}
	if !results[1].Matched {
		t.Fatal("Expected minimal segment to match")
	}
	if matchedMinimal == nil {
		t.Fatal("matchedMinimal is nil")
	}
	if matchedMinimal.Length() != 1 {
		t.Errorf("Expected matchedMinimal length 1, got %d", matchedMinimal.Length())
	}
	if minimalPrefix != 1 || minimalSuffix != 2 {
		t.Errorf("Expected minimalPrefix=1, minimalSuffix=2, got minimalPrefix=%d, minimalSuffix=%d", minimalPrefix, minimalSuffix)
	}

	// Test 2: Single bit nested bitstring
	singleBitInner, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(1)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create singleBitInner: %v", err)
	}
	if singleBitInner.Length() != 1 {
		t.Errorf("Expected singleBitInner length 1, got %d", singleBitInner.Length())
	}

	outerWithSingleBit, err := builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(7)).
		AddBitstring(singleBitInner).
		AddInteger(1, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create outerWithSingleBit: %v", err)
	}
	if outerWithSingleBit.Length() != 16 {
		t.Errorf("Expected outerWithSingleBit length 16, got %d", outerWithSingleBit.Length())
	}

	// Test matching single bit nested bitstring
	var singlePrefix, singleSuffix int
	var matchedSingleBit *bitstringpkg.BitString
	results2, err := matcher.NewMatcher().
		Integer(&singlePrefix, bitstringpkg.WithSize(7)).
		Bitstring(&matchedSingleBit, bitstringpkg.WithSize(1)).
		Integer(&singleSuffix, bitstringpkg.WithSize(8)).
		Match(outerWithSingleBit)
	if err != nil {
		t.Fatalf("Failed to match outerWithSingleBit: %v", err)
	}
	if !results2[1].Matched {
		t.Fatal("Expected single bit segment to match")
	}
	if matchedSingleBit == nil {
		t.Fatal("matchedSingleBit is nil")
	}
	if matchedSingleBit.Length() != 1 {
		t.Errorf("Expected matchedSingleBit length 1, got %d", matchedSingleBit.Length())
	}
	if singlePrefix != 0 || singleSuffix != 1 {
		t.Errorf("Expected singlePrefix=0, singleSuffix=1, got singlePrefix=%d, singleSuffix=%d", singlePrefix, singleSuffix)
	}

	// Test 3: Nested bitstring with non-aligned boundaries
	nonAlignedInner, err := builder.NewBuilder().
		AddInteger(0b10101010, bitstringpkg.WithSize(8)).
		AddInteger(0b11001100, bitstringpkg.WithSize(8)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create nonAlignedInner: %v", err)
	}

	nonAlignedOuter, err := builder.NewBuilder().
		AddInteger(0b111, bitstringpkg.WithSize(3)).
		AddBitstring(nonAlignedInner).
		AddInteger(0b000, bitstringpkg.WithSize(3)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create nonAlignedOuter: %v", err)
	}
	if nonAlignedOuter.Length() != 22 {
		t.Errorf("Expected nonAlignedOuter length 22, got %d", nonAlignedOuter.Length())
	}

	// Test matching non-aligned nested bitstring
	var nonAlignedPrefix, nonAlignedSuffix int
	var matchedNonAligned *bitstringpkg.BitString
	results3, err := matcher.NewMatcher().
		Integer(&nonAlignedPrefix, bitstringpkg.WithSize(3)).
		Bitstring(&matchedNonAligned, bitstringpkg.WithSize(16)).
		Integer(&nonAlignedSuffix, bitstringpkg.WithSize(3)).
		Match(nonAlignedOuter)
	if err != nil {
		t.Fatalf("Failed to match nonAlignedOuter: %v", err)
	}
	if !results3[1].Matched {
		t.Fatal("Expected non-aligned segment to match")
	}
	if matchedNonAligned == nil {
		t.Fatal("matchedNonAligned is nil")
	}
	if matchedNonAligned.Length() != 16 {
		t.Errorf("Expected matchedNonAligned length 16, got %d", matchedNonAligned.Length())
	}
	if nonAlignedPrefix != 0b111 || nonAlignedSuffix != 0b000 {
		t.Errorf("Expected nonAlignedPrefix=7, nonAlignedSuffix=0, got nonAlignedPrefix=%d, nonAlignedSuffix=%d", nonAlignedPrefix, nonAlignedSuffix)
	}
}

// TestBitstringNestedUnitAlignment tests unit alignment in nested structures
func TestBitstringNestedUnitAlignment(t *testing.T) {
	// Create bitstring with odd size
	oddSize, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(7)).
		Build()
	if err != nil {
		t.Fatalf("Failed to create oddSize: %v", err)
	}

	// Try to insert it with a size specification larger than available
	_, err = builder.NewBuilder().
		AddInteger(0, bitstringpkg.WithSize(8)).
		AddBitstring(oddSize, bitstringpkg.WithSize(8)). // Require 8 bits, but we only have 7
		AddInteger(1, bitstringpkg.WithSize(8)).
		Build()

	if err == nil {
		t.Fatal("Expected error for size mismatch, got nil")
	}
	if err.Error() == "" {
		t.Fatal("Expected error message, got empty string")
	}
	if !strings.Contains(err.Error(), "size mismatch") && !strings.Contains(err.Error(), "insufficient") && !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "less than") {
		t.Errorf("Expected error message to contain 'size mismatch', 'insufficient', 'invalid', or 'less than', got: %s", err.Error())
	}
}
