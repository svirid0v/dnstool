package main

const (
	domainsPath = "/opt/cache-domains"
	cacheDomain = "cache_domains.json"

	cacheConf  = "/etc/bind/cache.conf"
	namedConf  = "/etc/bind/named.conf.options"
	zonePath   = "/etc/bind/cache/"
	rpzZone    = zonePath + "rpz.db"
	customZone = zonePath + "custom.db"

	cacheConfTemplate = `	zone "cache.lancache.net" {
		type master;
		file "/etc/bind/cache/cache.lancache.net.db";
	};
	zone "rpz" {
		type master;
		file "/etc/bind/cache/rpz.db";
		allow-query { none; };
	};`

	rpzTemplate = `$TTL 60
@            IN    SOA  localhost. root.localhost.  (
                          2   ; serial 
                          3H  ; refresh 
                          1H  ; retry 
                          1W  ; expiry 
                          1H) ; minimum 
                  IN    NS    localhost.`
)
