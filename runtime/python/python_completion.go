package python

import (
	"fmt"
	"sort"
	"strings"

	"funterm/errors"
)

// Completion interface implementation

// GetModules returns available modules for the Python runtime
func (pr *PythonRuntime) GetModules() []string {
	if !pr.ready {
		return []string{}
	}

	modules := []string{
		"array", "asyncio", "bisect", "cmath", "collections", "contextlib", "copy",
		"dataclasses", "datetime", "enum", "fnmatch", "functools", "gc", "glob",
		"hashlib", "heapq", "hmac", "importlib", "inspect", "itertools", "json",
		"linecache", "macpath", "math", "multiprocessing", "ntpath", "numbers",
		"operator", "os", "os.path", "pathlib", "pkgutil", "posixpath", "pprint",
		"random", "re", "reprlib", "secrets", "shutil", "socket", "ssl",
		"statistics", "subprocess", "sys", "tempfile", "threading", "traceback",
		"types", "typing", "uuid", "warnings", "weakref",
		// Built-in types
		"dict", "list", "set", "str", "tuple",
	}
	sort.Strings(modules)
	return modules
}

// GetModuleFunctions returns available functions for a specific Python module
func (pr *PythonRuntime) GetModuleFunctions(module string) []string {
	if !pr.ready {
		return []string{}
	}

	switch module {
	case "math":
		functions := []string{
			"acos", "acosh", "asin", "asinh", "atan", "atan2", "atanh",
			"ceil", "comb", "copysign", "cos", "cosh", "degrees",
			"dist", "e", "erf", "erfc", "exp", "expm1", "fabs",
			"factorial", "floor", "fmod", "frexp", "fsum", "gamma",
			"gcd", "hypot", "inf", "isclose", "isfinite", "isinf",
			"isnan", "isqrt", "lcm", "ldexp", "lgamma", "log",
			"log10", "log1p", "log2", "modf", "nan", "nextafter",
			"perm", "pi", "pow", "prod", "radians", "remainder",
			"sin", "sinh", "sqrt", "tan", "tanh", "tau",
			"trunc", "ulp",
		}
		sort.Strings(functions)
		return functions
	case "str":
		functions := []string{
			"capitalize", "casefold", "center", "count", "encode",
			"endswith", "expandtabs", "find", "format", "format_map",
			"index", "isalnum", "isalpha", "isascii", "isdecimal",
			"isdigit", "isidentifier", "islower", "isnumeric", "isprintable",
			"isspace", "istitle", "isupper", "join", "ljust",
			"lower", "lstrip", "maketrans", "partition", "removeprefix",
			"removesuffix", "replace", "rfind", "rindex", "rjust",
			"rpartition", "rsplit", "rstrip", "split", "splitlines",
			"startswith", "strip", "swapcase", "title", "translate",
			"upper", "zfill",
		}
		sort.Strings(functions)
		return functions
	case "list":
		functions := []string{
			"append", "clear", "copy", "count", "extend",
			"index", "insert", "pop", "remove", "reverse", "sort",
		}
		sort.Strings(functions)
		return functions
	case "dict":
		functions := []string{
			"clear", "copy", "fromkeys", "get", "items",
			"keys", "pop", "popitem", "setdefault", "update", "values",
		}
		sort.Strings(functions)
		return functions
	case "os":
		functions := []string{
			"abort", "access", "chdir", "chmod", "chown", "chroot",
			"close", "closerange", "confstr", "confstr_names", "ctermid",
			"dup", "dup2", "environ", "error", "execl", "execle",
			"execlp", "execlpe", "execv", "execve", "execvp", "execvpe",
			"exit", "fchdir", "fchmod", "fchown", "fdatasync", "fdopen",
			"fork", "forkpty", "fpathconf", "fstat", "fstatvfs", "fsync",
			"ftruncate", "getcwd", "getcwdb", "getegid", "getenv", "geteuid",
			"getgid", "getgrouplist", "getgroups", "getloadavg", "getlogin",
			"getpgid", "getpgrp", "getpid", "getppid", "getpriority",
			"getsid", "getuid", "getxattr", "initgroups", "isatty", "kill",
			"killpg", "lchown", "link", "listdir", "listxattr", "lockf",
			"lseek", "lstat", "major", "makedev", "makedirs", "minor",
			"mkdir", "mkfifo", "mknod", "name", "nice", "open", "openpty",
			"pathconf", "pathconf_names", "pathsep", "pidfd_open", "pipe",
			"popen", "posix_fadvise", "posix_fallocate", "posix_spawn",
			"posix_spawnp", "pread", "preadv", "putenv", "pwrite", "pwritev",
			"read", "readlink", "readv", "remove", "removedirs", "rename",
			"renames", "replace", "rmdir", "scandir", "sched_getaffinity",
			"sched_getparam", "sched_getscheduler", "sched_param",
			"sched_rr_get_interval", "sched_setaffinity", "sched_setparam",
			"sched_setscheduler", "sched_yield", "sendfile", "setegid",
			"seteuid", "setgid", "setgroups", "setpgid", "setpgrp",
			"setpriority", "setregid", "setreuid", "setsid", "setuid",
			"setxattr", "spawnl", "spawnle", "spawnlp", "spawnlpe", "spawnv",
			"spawnve", "spawnvp", "spawnvpe", "stat", "statvfs",
			"strerror", "supports_bytes_environ", "supports_dir_fd",
			"supports_effective_ids", "supports_fd", "supports_follow_symlinks",
			"symlink", "sysconf", "sysconf_names", "system", "tcgetpgrp",
			"tcsetpgrp", "terminal_size", "times", "truncate", "umask",
			"uname", "unlink", "unsetenv", "urandom", "utime", "wait",
			"wait3", "wait4", "waitid", "waitpid", "write", "writev",
		}
		sort.Strings(functions)
		return functions
	case "sys":
		functions := []string{
			"abiflags", "addaudithook", "audit", "base_exec_prefix",
			"base_prefix", "byteorder", "builtin_module_names",
			"call_tracing", "copyright", "displayhook", "dont_write_bytecode",
			"exc_info", "excepthook", "exec_prefix", "executable", "exit",
			"flags", "float_info", "float_repr_style", "get_asyncgen_hooks",
			"get_coroutine_origin_tracking_depth", "getallocatedblocks",
			"getdefaultencoding", "getfilesystemencodeerrors", "getfilesystemencoding",
			"getprofile", "getrecursionlimit", "getrefcount", "getsizeof",
			"getswitchinterval", "gettrace", "hash_info", "hexversion",
			"implementation", "int_info", "intern", "is_finalizing",
			"maxsize", "maxunicode", "meta_path", "modules", "path",
			"path_hooks", "path_importer_cache", "platform", "prefix",
			"ps1", "ps2", "pycache_prefix", "set_asyncgen_hooks",
			"set_coroutine_origin_tracking_depth", "setprofile", "setrecursionlimit",
			"setswitchinterval", "settrace", "stderr", "stdin", "stdout",
			"thread_info", "version", "version_info", "warnoptions",
		}
		sort.Strings(functions)
		return functions
	case "json":
		functions := []string{
			"dump", "dumps", "load", "loads",
			"JSONDecoder", "JSONEncoder",
		}
		sort.Strings(functions)
		return functions
	case "datetime":
		functions := []string{
			"date", "datetime", "time", "timedelta", "timezone",
			"MINYEAR", "MAXYEAR",
		}
		sort.Strings(functions)
		return functions
	case "random":
		functions := []string{
			"choice", "choices", "random", "randint", "randrange",
			"sample", "shuffle", "uniform", "betavariate", "expovariate",
			"gammavariate", "gauss", "lognormvariate", "normalvariate",
			"paretovariate", "vonmisesvariate", "weibullvariate",
			"getstate", "setstate", "getrandbits", "seed",
		}
		sort.Strings(functions)
		return functions
	case "re":
		functions := []string{
			"compile", "escape", "findall", "finditer", "fullmatch",
			"match", "search", "split", "sub", "subn", "template",
			"purge", "error", "A", "I", "L", "M", "S", "X", "U",
			"ASCII", "IGNORECASE", "LOCALE", "MULTILINE", "DOTALL",
			"VERBOSE", "UNICODE",
		}
		sort.Strings(functions)
		return functions
	case "collections":
		functions := []string{
			"namedtuple", "deque", "ChainMap", "Counter", "OrderedDict",
			"defaultdict", "UserDict", "UserList", "UserString",
		}
		sort.Strings(functions)
		return functions
	case "itertools":
		functions := []string{
			"accumulate", "chain", "combinations", "combinations_with_replacement",
			"compress", "count", "cycle", "dropwhile", "filterfalse",
			"groupby", "islice", "permutations", "product", "repeat",
			"starmap", "takewhile", "tee", "zip_longest",
		}
		sort.Strings(functions)
		return functions
	case "functools":
		functions := []string{
			"reduce", "partial", "lru_cache", "singledispatch",
			"total_ordering", "wraps", "update_wrapper", "cached_property",
			"cmp_to_key",
		}
		sort.Strings(functions)
		return functions
	default:
		return []string{}
	}
}

// GetFunctionSignature returns the signature of a function in a module
func (pr *PythonRuntime) GetFunctionSignature(module, function string) (string, error) {
	if !pr.ready {
		return "", errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Define function signatures for common Python functions
	signatures := map[string]map[string]string{
		"math": {
			"acos":    "acos(x) -> float",
			"acosh":   "acosh(x) -> float",
			"asin":    "asin(x) -> float",
			"asinh":   "asinh(x) -> float",
			"atan":    "atan(x) -> float",
			"atan2":   "atan2(y, x) -> float",
			"atanh":   "atanh(x) -> float",
			"ceil":    "ceil(x) -> int",
			"cos":     "cos(x) -> float",
			"cosh":    "cosh(x) -> float",
			"degrees": "degrees(x) -> float",
			"exp":     "exp(x) -> float",
			"floor":   "floor(x) -> int",
			"log":     "log(x[, base]) -> float",
			"log10":   "log10(x) -> float",
			"pow":     "pow(x, y) -> float",
			"radians": "radians(x) -> float",
			"sin":     "sin(x) -> float",
			"sinh":    "sinh(x) -> float",
			"sqrt":    "sqrt(x) -> float",
			"tan":     "tan(x) -> float",
			"tanh":    "tanh(x) -> float",
		},
		"str": {
			"capitalize": "capitalize() -> str",
			"casefold":   "casefold() -> str",
			"center":     "center(width[, fillchar]) -> str",
			"count":      "count(sub[, start[, end]]) -> int",
			"endswith":   "endswith(suffix[, start[, end]]) -> bool",
			"find":       "find(sub[, start[, end]]) -> int",
			"format":     "format(*args, **kwargs) -> str",
			"index":      "index(sub[, start[, end]]) -> int",
			"isalnum":    "isalnum() -> bool",
			"isalpha":    "isalpha() -> bool",
			"isdigit":    "isdigit() -> bool",
			"islower":    "islower() -> bool",
			"isspace":    "isspace() -> bool",
			"istitle":    "istitle() -> bool",
			"isupper":    "isupper() -> bool",
			"join":       "join(iterable) -> str",
			"lower":      "lower() -> str",
			"lstrip":     "lstrip([chars]) -> str",
			"replace":    "replace(old, new[, count]) -> str",
			"rfind":      "rfind(sub[, start[, end]]) -> int",
			"rindex":     "rindex(sub[, start[, end]]) -> int",
			"rstrip":     "rstrip([chars]) -> str",
			"split":      "split([sep[, maxsplit]]) -> list",
			"startswith": "startswith(prefix[, start[, end]]) -> bool",
			"strip":      "strip([chars]) -> str",
			"swapcase":   "swapcase() -> str",
			"title":      "title() -> str",
			"upper":      "upper() -> str",
		},
		"list": {
			"append":  "append(object) -> None",
			"clear":   "clear() -> None",
			"copy":    "copy() -> list",
			"count":   "count(value) -> int",
			"extend":  "extend(iterable) -> None",
			"index":   "index(value[, start[, stop]]) -> int",
			"insert":  "insert(index, object) -> None",
			"pop":     "pop([index]) -> item",
			"remove":  "remove(value) -> None",
			"reverse": "reverse() -> None",
			"sort":    "sort(key=None, reverse=False) -> None",
		},
		"dict": {
			"clear":      "clear() -> None",
			"copy":       "copy() -> dict",
			"fromkeys":   "fromkeys(iterable[, value]) -> dict",
			"get":        "get(key[, default]) -> value",
			"items":      "items() -> dict_items",
			"keys":       "keys() -> dict_keys",
			"pop":        "pop(key[, default]) -> value",
			"popitem":    "popitem() -> (key, value)",
			"setdefault": "setdefault(key[, default]) -> value",
			"update":     "update([other]) -> None",
			"values":     "values() -> dict_values",
		},
	}

	if moduleSignatures, ok := signatures[module]; ok {
		if signature, ok := moduleSignatures[function]; ok {
			return signature, nil
		}
	}

	return "", errors.NewRuntimeError("python", "FUNCTION_NOT_FOUND", fmt.Sprintf("function '%s.%s' not found", module, function))
}

// GetGlobalVariables returns available global variables
func (pr *PythonRuntime) GetGlobalVariables() []string {
	if !pr.ready {
		return []string{}
	}

	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	var variables []string
	for name := range pr.variables {
		// Skip internal variables that start with "_" but not "__"
		// This allows important global variables like "__name__", "__file__", etc.
		if !strings.HasPrefix(name, "_") || strings.HasPrefix(name, "__") {
			variables = append(variables, name)
		}
	}

	// Add common Python built-ins
	builtins := []string{
		"abs", "all", "any", "ascii", "bin", "bool", "breakpoint",
		"bytearray", "bytes", "callable", "chr", "classmethod",
		"compile", "complex", "delattr", "dict", "dir", "divmod",
		"enumerate", "eval", "exec", "filter", "float", "format",
		"frozenset", "getattr", "globals", "hasattr", "hash",
		"help", "hex", "id", "input", "int", "isinstance",
		"issubclass", "iter", "len", "list", "locals", "map",
		"max", "memoryview", "min", "next", "object", "oct",
		"open", "ord", "pow", "print", "property", "range",
		"repr", "reversed", "round", "set", "setattr", "slice",
		"sorted", "staticmethod", "str", "sum", "super", "tuple",
		"type", "vars", "zip", "__import__",
		// Add important Python global variables
		"__name__", "__file__", "__doc__", "__package__", "__loader__", "__spec__", "__builtins__",
	}

	variables = append(variables, builtins...)
	return variables
}

// GetCompletionSuggestions returns completion suggestions for a given input
func (pr *PythonRuntime) GetCompletionSuggestions(input string) []string {
	if !pr.ready {
		return []string{}
	}

	input = strings.TrimSpace(input)
	if input == "" {
		// Return all modules and global variables
		suggestions := pr.GetModules()
		suggestions = append(suggestions, pr.GetGlobalVariables()...)
		return suggestions
	}

	// Check if input contains a dot (module.function)
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts) == 2 {
			module := parts[0]
			prefix := parts[1]

			// Check if it's a valid module
			if pr.isModule(module) {
				functions := pr.GetModuleFunctions(module)
				var suggestions []string
				for _, fn := range functions {
					if strings.HasPrefix(fn, prefix) {
						suggestions = append(suggestions, fmt.Sprintf("%s.%s", module, fn))
					}
				}
				return suggestions
			}
		}
	} else {
		// Check for module or global variable completion
		var suggestions []string

		// Check modules
		modules := pr.GetModules()
		for _, module := range modules {
			if strings.HasPrefix(module, input) {
				suggestions = append(suggestions, module+".")
			}
		}

		// Check global variables
		variables := pr.GetGlobalVariables()
		for _, variable := range variables {
			if strings.HasPrefix(variable, input) {
				suggestions = append(suggestions, variable)
			}
		}

		return suggestions
	}

	return []string{}
}

// isModule checks if a name is a valid Python module
func (pr *PythonRuntime) isModule(name string) bool {
	modules := pr.GetModules()
	for _, module := range modules {
		if module == name {
			return true
		}
	}
	return false
}
