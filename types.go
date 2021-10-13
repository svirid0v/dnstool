package main

type CacheFile struct {
	CacheDomains []struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		DomainFiles  []string `json:"domain_files"`
		Notes        string   `json:"notes,omitempty"`
		MixedContent bool     `json:"mixed_content,omitempty"`
	} `json:"cache_domains"`
}
