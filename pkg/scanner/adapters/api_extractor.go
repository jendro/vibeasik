package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vibeasik/vibeasik/pkg/util"
)

var (
	rxQueryParam   = regexp.MustCompile(`\?.*$`)
	rxBaseVar      = regexp.MustCompile(`(?i)\$\{[^}]*(?:base|api|url|host|origin|env)[^}]*\}`)
	rxDynamicVar   = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)
	rxRemainingVar = regexp.MustCompile(`\$\{[^}]+\}`)
	rxDoubleSlash  = regexp.MustCompile(`/{2,}`)

	rxFnAssign   = regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s*)?\([^)]*\)\s*(?::\s*[^=]+)?\s*=>`)
	rxFuncDef    = regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_]+)\s*\([^)]*\)`)
	rxObjMethod  = regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*:\s*(?:async\s*)?\([^)]*\)\s*=>`)
	rxApiPath    = regexp.MustCompile(`['"` + "`" + `](?:(?:\$\{[^}]+\})|\/api|\bhttp[s]?:\/\/)[^'"` + "`" + `\s]*['"` + "`" + `]`)
	rxMethod     = regexp.MustCompile(`(?i)\b(?:method\s*:\s*['"](GET|POST|PUT|DELETE|PATCH)['"]|\.(get|post|put|delete|patch)\b)`)
	rxDirectReq  = regexp.MustCompile(`(?i)(?:\$fetch|fetch|axios(?:\.(get|post|put|delete|patch))?|useFetch|useAsyncData)\s*(?:<[^>]+>)?\s*\(\s*['"` + "`" + `]([^'"` + "`" + `]+)['"` + "`" + `]`)
	rxImportPath = regexp.MustCompile(`import\s+[a-zA-Z0-9_{},\s*]+\s+from\s+['"]([^'"]+)['"]`)
)

// CleanEndpointURL normalizes URL paths by stripping base variables, preserving dynamic route variables, and standardizing slashes
func CleanEndpointURL(rawUrl string) string {
	clean := strings.Trim(rawUrl, `'"`+"`")
	clean = rxQueryParam.ReplaceAllString(clean, "")
	clean = rxBaseVar.ReplaceAllString(clean, "")
	clean = rxDynamicVar.ReplaceAllString(clean, ":$1")
	clean = rxRemainingVar.ReplaceAllString(clean, ":param")
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	clean = rxDoubleSlash.ReplaceAllString(clean, "/")
	clean = strings.TrimRight(clean, "/")
	if clean == "" {
		clean = "/"
	}
	return clean
}

// BuildFrontendApiFunctionMap scans composables, services, and api client files in a frontend project
func BuildFrontendApiFunctionMap(dir string) map[string]string {
	fnMap := make(map[string]string)
	candidateDirs := []string{
		"composables",
		"app/composables",
		"services",
		"src/services",
		"api",
		"src/api",
		"lib/api",
		"src/lib/api",
		"utils",
		"app/utils",
		"src/utils",
	}

	for _, cand := range candidateDirs {
		fullCand := filepath.Join(dir, cand)
		info, err := os.Stat(fullCand)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(fullCand, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".ts" && ext != ".js" && ext != ".tsx" && ext != ".jsx" {
				return nil
			}

			contentBytes, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			ParseApiFunctionsFromSource(string(contentBytes), fnMap)
			return nil
		})
	}

	return fnMap
}

// apiFunctionMatches mengumpulkan posisi deklarasi fungsi (arrow, function, method objek) terurut.
func apiFunctionMatches(content string) [][]int {
	allMatches := append(rxFnAssign.FindAllStringSubmatchIndex(content, -1), rxFuncDef.FindAllStringSubmatchIndex(content, -1)...)
	allMatches = append(allMatches, rxObjMethod.FindAllStringSubmatchIndex(content, -1)...)
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i][0] < allMatches[j][0]
	})
	return allMatches
}

// firstApiPath mencari literal path API pertama dalam blok fungsi.
func firstApiPath(block string) string {
	for _, urlMatch := range rxApiPath.FindAllString(block, -1) {
		cleanUrl := strings.Trim(urlMatch, `'"`+"`")
		if strings.Contains(cleanUrl, "/api") || strings.Contains(cleanUrl, "${") || strings.HasPrefix(cleanUrl, "/") {
			return cleanUrl
		}
	}
	return ""
}

// blockMethod mengambil metode HTTP pertama yang tertulis dalam blok; default GET.
func blockMethod(block string) string {
	if mm := rxMethod.FindStringSubmatch(block); len(mm) > 0 {
		for k := 1; k < len(mm); k++ {
			if mm[k] != "" {
				return strings.ToUpper(mm[k])
			}
		}
	}
	return "GET"
}

// ParseApiFunctionsFromSource inspects a JS/TS source file to extract API helper functions
func ParseApiFunctionsFromSource(content string, fnMap map[string]string) {
	allMatches := apiFunctionMatches(content)
	for i, m := range allMatches {
		fnName := content[m[2]:m[3]]
		if fnName == "useApi" || fnName == "useAuth" {
			continue
		}

		startPos := m[0]
		endPos := len(content)
		if i+1 < len(allMatches) {
			endPos = allMatches[i+1][0]
		}
		if endPos-startPos > 1200 {
			endPos = startPos + 1200
		}
		block := content[startPos:endPos]

		foundPath := firstApiPath(block)
		if foundPath == "" {
			continue
		}

		lower := strings.ToLower(foundPath)
		isApiBase := strings.Contains(lower, "base") || strings.Contains(lower, "api") || strings.Contains(lower, "url")

		cleanPath := CleanEndpointURL(foundPath)
		if isApiBase && !strings.HasPrefix(cleanPath, "/api") {
			cleanPath = "/api" + cleanPath
		}
		fnMap[fnName] = fmt.Sprintf("%s %s", blockMethod(block), cleanPath)
	}
}

// collectCalledHelpers menambahkan endpoint fungsi helper (apiFuncMap) yang dipakai dalam konten.
func collectCalledHelpers(content string, apiFuncMap map[string]string, reqSet map[string]bool) {
	for fnName, endpoint := range apiFuncMap {
		pattern := `\b` + regexp.QuoteMeta(fnName) + `\b`
		if matched, _ := regexp.MatchString(pattern, content); matched {
			reqSet[endpoint] = true
		}
	}
}

// collectDirectRequests menambahkan $fetch, fetch, axios, useFetch, useAsyncData langsung dalam konten.
func collectDirectRequests(content string, reqSet map[string]bool) {
	for _, m := range rxDirectReq.FindAllStringSubmatch(content, -1) {
		if len(m) < 3 {
			continue
		}
		httpMethod := "GET"
		if m[1] != "" {
			httpMethod = strings.ToUpper(m[1])
		}
		urlPath := m[2]
		if strings.Contains(urlPath, "/api") || strings.HasPrefix(urlPath, "/") || strings.Contains(urlPath, "${") {
			cleanPath := CleanEndpointURL(urlPath)
			if strings.Contains(strings.ToLower(urlPath), "api") && !strings.HasPrefix(cleanPath, "/api") {
				cleanPath = "/api" + cleanPath
			}
			reqSet[fmt.Sprintf("%s %s", httpMethod, cleanPath)] = true
		}
	}
}

var importExts = []string{"", ".vue", ".tsx", ".jsx", ".ts", ".js"}

// firstExistingWithExt mengembalikan candidate+ekstensi pertama yang ada, atau "".
func firstExistingWithExt(candidate string) string {
	for _, ext := range importExts {
		if util.FileExists(candidate + ext) {
			return candidate + ext
		}
	}
	return ""
}

// resolveImport menerjemahkan path import komponen (relatif atau alias @/ ~/) menjadi berkas nyata, atau "".
func resolveImport(impPath, pagePath, dir string) string {
	if !strings.HasPrefix(impPath, "@/") && !strings.HasPrefix(impPath, "~/") {
		return firstExistingWithExt(filepath.Join(filepath.Dir(pagePath), impPath))
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(impPath, "@/"), "~/")
	for _, tryPrefix := range []string{"app/", "src/", ""} {
		if resolved := firstExistingWithExt(filepath.Join(dir, tryPrefix+rel)); resolved != "" {
			return resolved
		}
	}
	return ""
}

func isLocalImport(impPath string) bool {
	return strings.HasPrefix(impPath, ".") || strings.HasPrefix(impPath, "@/") || strings.HasPrefix(impPath, "~/")
}

// ExtractPageTargetRequests parses a page component and detects all outgoing API requests
func ExtractPageTargetRequests(pagePath string, pageContent string, dir string, apiFuncMap map[string]string) []string {
	reqSet := make(map[string]bool)

	// 1. Check helper functions from apiFuncMap used in page
	collectCalledHelpers(pageContent, apiFuncMap, reqSet)

	// 2. Direct $fetch, fetch, axios, useFetch, useAsyncData
	collectDirectRequests(pageContent, reqSet)

	// 3. Child components API calls (scan components imported in this page)
	for _, match := range rxImportPath.FindAllStringSubmatch(pageContent, -1) {
		if len(match) < 2 || !isLocalImport(match[1]) {
			continue
		}
		resolvedPath := resolveImport(match[1], pagePath, dir)
		if resolvedPath == "" || resolvedPath == pagePath {
			continue
		}
		childBytes, err := os.ReadFile(resolvedPath)
		if err != nil {
			continue
		}
		childContent := string(childBytes)
		collectCalledHelpers(childContent, apiFuncMap, reqSet)
		collectDirectRequests(childContent, reqSet)
	}

	var results []string
	for r := range reqSet {
		results = append(results, r)
	}
	sort.Strings(results)
	return results
}
