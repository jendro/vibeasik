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

// ParseApiFunctionsFromSource inspects a JS/TS source file to extract API helper functions
func ParseApiFunctionsFromSource(content string, fnMap map[string]string) {
	matches := rxFnAssign.FindAllStringSubmatchIndex(content, -1)
	funcMatches := rxFuncDef.FindAllStringSubmatchIndex(content, -1)
	objMatches := rxObjMethod.FindAllStringSubmatchIndex(content, -1)

	allMatches := append(matches, funcMatches...)
	allMatches = append(allMatches, objMatches...)
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i][0] < allMatches[j][0]
	})

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

		foundPath := ""
		for _, urlMatch := range rxApiPath.FindAllString(block, -1) {
			cleanUrl := strings.Trim(urlMatch, `'"`+"`")
			if strings.Contains(cleanUrl, "/api") || strings.Contains(cleanUrl, "${") || strings.HasPrefix(cleanUrl, "/") {
				foundPath = cleanUrl
				break
			}
		}

		if foundPath == "" {
			continue
		}

		isApiBase := strings.Contains(strings.ToLower(foundPath), "base") ||
			strings.Contains(strings.ToLower(foundPath), "api") ||
			strings.Contains(strings.ToLower(foundPath), "url")

		cleanPath := CleanEndpointURL(foundPath)
		if isApiBase && !strings.HasPrefix(cleanPath, "/api") {
			cleanPath = "/api" + cleanPath
		}

		method := "GET"
		if mm := rxMethod.FindStringSubmatch(block); len(mm) > 0 {
			for k := 1; k < len(mm); k++ {
				if mm[k] != "" {
					method = strings.ToUpper(mm[k])
					break
				}
			}
		}

		fnMap[fnName] = fmt.Sprintf("%s %s", method, cleanPath)
	}
}

// ExtractPageTargetRequests parses a page component and detects all outgoing API requests
func ExtractPageTargetRequests(pagePath string, pageContent string, dir string, apiFuncMap map[string]string) []string {
	reqSet := make(map[string]bool)

	// 1. Check helper functions from apiFuncMap used in page
	for fnName, endpoint := range apiFuncMap {
		pattern := `\b` + regexp.QuoteMeta(fnName) + `\b`
		if matched, _ := regexp.MatchString(pattern, pageContent); matched {
			reqSet[endpoint] = true
		}
	}

	// 2. Direct $fetch, fetch, axios, useFetch, useAsyncData
	for _, m := range rxDirectReq.FindAllStringSubmatch(pageContent, -1) {
		if len(m) >= 3 {
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

	// 3. Child components API calls (scan components imported in this page)
	for _, match := range rxImportPath.FindAllStringSubmatch(pageContent, -1) {
		if len(match) < 2 {
			continue
		}
		impPath := match[1]
		if !strings.HasPrefix(impPath, ".") && !strings.HasPrefix(impPath, "@/") && !strings.HasPrefix(impPath, "~/") {
			continue
		}

		resolvedPath := ""
		if strings.HasPrefix(impPath, "@/") || strings.HasPrefix(impPath, "~/") {
			rel := strings.TrimPrefix(strings.TrimPrefix(impPath, "@/"), "~/")
			for _, tryPrefix := range []string{"app/", "src/", ""} {
				candidate := filepath.Join(dir, tryPrefix+rel)
				for _, ext := range []string{"", ".vue", ".tsx", ".jsx", ".ts", ".js"} {
					if util.FileExists(candidate + ext) {
						resolvedPath = candidate + ext
						break
					}
				}
				if resolvedPath != "" {
					break
				}
			}
		} else {
			dirOfPage := filepath.Dir(pagePath)
			candidate := filepath.Join(dirOfPage, impPath)
			for _, ext := range []string{"", ".vue", ".tsx", ".jsx", ".ts", ".js"} {
				if util.FileExists(candidate + ext) {
					resolvedPath = candidate + ext
					break
				}
			}
		}

		if resolvedPath != "" && resolvedPath != pagePath {
			childBytes, err := os.ReadFile(resolvedPath)
			if err == nil {
				childContent := string(childBytes)
				for fnName, endpoint := range apiFuncMap {
					pattern := `\b` + regexp.QuoteMeta(fnName) + `\b`
					if matched, _ := regexp.MatchString(pattern, childContent); matched {
						reqSet[endpoint] = true
					}
				}
				for _, m := range rxDirectReq.FindAllStringSubmatch(childContent, -1) {
					if len(m) >= 3 {
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
			}
		}
	}

	var results []string
	for r := range reqSet {
		results = append(results, r)
	}
	sort.Strings(results)
	return results
}
