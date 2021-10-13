package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	logger "log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var log = logger.New(os.Stdout, "", 0)

func main() {
	useGenericCache := "false"
	if os.Getenv("USE_GENERIC_CACHE") != "" {
		useGenericCache = os.Getenv("USE_GENERIC_CACHE")
	}

	lancacheDNSDomain := "cache.lancache.net"
	if os.Getenv("LANCACHE_DNSDOMAIN") != "cache.lancache.net" {
		lancacheDNSDomain = os.Getenv("LANCACHE_DNSDOMAIN")
	}

	cacheZone := zonePath + lancacheDNSDomain + ".db"

	upstreamDNS := "8.8.8.8"
	if os.Getenv("UPSTREAM_DNS") != "8.8.8.8" {
		upstreamDNS = os.Getenv("UPSTREAM_DNS")
	}

	adjustedDNS := strings.ReplaceAll(upstreamDNS, ";", " ")
	cleanDNS := strings.Split(adjustedDNS, " ")

	for _, s := range cleanDNS {
		if err := net.ParseIP(s); err == nil {
			log.Fatalf("IP address: %s is not valid", s)
		}
	}

	if err := writeResolverConfiguration(cleanDNS); err != nil {
		log.Fatal(err)
	}

	if err := bootstrapDNS(); err != nil {
		log.Fatal(err)
	}

	cacheIP := os.Getenv("LANCACHE_IP")
	if err := checkGenericCache(useGenericCache, cacheIP); err != nil {
		log.Fatal(err)
	}

	if err := generateConfiguration(useGenericCache, lancacheDNSDomain, cacheIP, cacheZone); err != nil {
		log.Fatal(err)
	}
}

func writeResolverConfiguration(dns []string) error {
	log.Print("Configuring /etc/resolv.conf to stop from looping to ourself")

	f, err := os.Create("/etc/resolv.conf")
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = fmt.Fprintln(f, "# Lancache dns config"); err != nil {
		return err
	}

	for _, d := range dns {
		if _, err = fmt.Fprintln(f, "nameserver "+d); err != nil {
			return err
		}
	}

	return nil
}

func bootstrapDNS() error {
	cacheDomainsRepo := os.Getenv("CACHE_DOMAINS_REPO")
	cacheDomainsBranch := os.Getenv("CACHE_DOMAINS_BRANCH")

	log.Printf("Bootstrapping Lancache-DNS from %s", cacheDomainsRepo)

	if _, err := os.Stat(domainsPath + "/.git"); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", cacheDomainsRepo, ".")
		cmd.Dir = domainsPath

		cmd.Env = append(os.Environ(),
			"GIT_SSH_COMMAND=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no")

		if err = cmd.Run(); err != nil {
			return err
		}
	}

	if os.Getenv("NOFETCH") != "true" {
		cmd := exec.Command("git", "remote", "set-url", "origin", cacheDomainsRepo)
		cmd.Dir = domainsPath
		_ = cmd.Run()

		cmd = exec.Command("git", "fetch", "origin")
		cmd.Dir = domainsPath

		if err := cmd.Run(); err != nil {
			log.Print("Failed to update from remote, using local copy of cache_domains")
		}

		cmd = exec.Command("git", "reset", "--hard", "origin/"+cacheDomainsBranch)
		cmd.Dir = domainsPath
		_ = cmd.Run()
	}

	return nil
}

func checkGenericCache(useGenericCache, cacheIP string) error {
	if useGenericCache == "true" {
		if cacheIP == "" {
			return fmt.Errorf("If you are using USE_GENERIC_CACHE then you must set LANCACHE_IP")
		}
	} else {
		if cacheIP != "" {
			return fmt.Errorf("If you are using LANCACHE_IP then you must set USE_GENERIC_CACHE=true")
		}
	}

	return nil
}

func generateConfiguration(useGenericCache, lancacheDNSDomain, cacheIP, cacheZone string) error {
	if useGenericCache == "true" {
		log.Print("")
		log.Print("----------------------------------------------------------------------")
		log.Printf("Using Generic Server: %s", cacheIP)
		log.Printf("Make sure you are using a monolithic cache or load balancer at %s", cacheIP)
		log.Print("----------------------------------------------------------------------")
		log.Print("")
	}

	if err := generateCacheConf(); err != nil {
		return err
	}

	if err := generateCacheZone(lancacheDNSDomain, cacheZone); err != nil {
		return err
	}

	if err := generateRPZZone(); err != nil {
		return err
	}

	services, serviceFiles, err := identifyServices()
	if err != nil {
		return err
	}

	if err := checkService(useGenericCache, cacheIP, cacheZone, lancacheDNSDomain, services, serviceFiles); err != nil {
		return err
	}

	log.Print(`
 --- 

`)

	if err := finaliseConfiguration(); err != nil {
		return err
	}

	return nil
}

func generateCacheConf() error {
	f, err := os.Create(cacheConf)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = fmt.Fprintln(f, cacheConfTemplate); err != nil {
		return err
	}

	return nil
}

func generateCacheZone(lancacheDNSDomain, cacheZone string) error {
	f, err := os.Create(cacheZone)
	if err != nil {
		return err
	}

	defer f.Close()

	now := time.Now()
	if _, err = fmt.Fprintln(f, `$ORIGIN `+lancacheDNSDomain+`. 
$TTL    600
@       IN  SOA localhost. dns.lancache.net. (
             `+fmt.Sprint(now.Unix())+`
             604800	
             600
             600
             600 )
@       IN  NS  localhost.`); err != nil {
		return err
	}

	return nil
}

func generateRPZZone() error {
	f, err := os.Create(rpzZone)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = fmt.Fprintln(f, rpzTemplate); err != nil {
		return err
	}

	return nil
}

func identifyServices() ([]string, []string, error) {
	f, err := ioutil.ReadFile(domainsPath + "/" + cacheDomain)
	if err != nil {
		return nil, nil, err
	}

	var cacheData CacheFile

	err = json.Unmarshal(f, &cacheData)
	if err != nil {
		return nil, nil, err
	}

	serviceMap := make([]string, 0)
	serviceFileMap := make([]string, 0)

	for _, services := range cacheData.CacheDomains {
		service := services.Name
		serviceMap = append(serviceMap, service)
		serviceFileMap = append(serviceFileMap, services.DomainFiles[0])
	}

	return serviceMap, serviceFileMap, nil
}

func checkService(genericCache, cacheIP, cacheZone, lancacheDNSDomain string, services, serviceFiles []string) error {
	for i, service := range services {
		log.Printf("Processing service: %s", service)

		if err := generateService(genericCache, cacheIP, cacheZone, lancacheDNSDomain, service, serviceFiles[i]); err != nil {
			return err
		}
	}

	return nil
}

func generateService(genericCache, cacheIP, cacheZone, lancacheDNSDomain, service, serviceFile string) error {
	enabled := false
	populate := false
	ip := ""

	service = strings.ToUpper(service)
	if genericCache == "true" {
		if os.Getenv("DISABLE_"+service) != "true" {
			enabled = true
		}
	} else {
		log.Printf("Testing for presence of %sCACHE_IP", service)
		if _, ok := os.LookupEnv(service + "CACHE_IP"); ok {
			enabled = true
		}
	}

	if enabled {
		if os.Getenv(service+"CACHE_IP") != "" {
			ip = os.Getenv(service + "CACHE_IP")
		} else {
			ip = cacheIP
		}

		if ip != "" {
			log.Printf("Enabling service with IP(s): %s", ip)

			service = strings.ToLower(service)

			f, err := os.OpenFile(rpzZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}

			defer f.Close()

			if _, err = fmt.Fprintln(f, `;## `+service); err != nil {
				return err
			}

			cleanIP := strings.Split(ip, " ")
			for _, ip := range cleanIP {
				c, err := os.OpenFile(cacheZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}

				defer c.Close()

				if _, err = fmt.Fprintln(c, service+` IN A `+ip+`;`); err != nil {
					return err
				}

				r, err := os.OpenFile(rpzZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}

				defer r.Close()

				revIP := reverseIPv4(strings.Split(ip, "."))
				if _, err = fmt.Fprintln(r, `32.`+revIP+`.rpz-client-ip      CNAME rpz-passthru.;`); err != nil {
					return err
				}

				populate = true
			}
		} else {
			return fmt.Errorf("Could not find IP for requested service: %s", service)
		}
	} else {
		log.Printf("Skipping service: %s", strings.ToLower(service))
	}

	if populate {
		if err := generateDomains(serviceFile, lancacheDNSDomain, service); err != nil {
			return err
		}
	}

	return nil
}

func generateDomains(serviceFile, lancacheDNSDomain, service string) error {
	f, err := os.Open(domainsPath + "/" + serviceFile)
	if err != nil {
		return err
	}

	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}

		if strings.HasPrefix(string(line), "#") {
			continue
		}

		r, err := os.OpenFile(rpzZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		defer r.Close()

		if _, err = fmt.Fprintln(r, string(line)+` IN CNAME `+service+`.`+lancacheDNSDomain+`.;`); err != nil {
			return err
		}
	}

	return nil
}

func finaliseConfiguration() error {
	if ip := os.Getenv("PASSTHRU_IPS"); ip != "" {
		cleanIP := strings.Split(ip, " ")
		for _, ip := range cleanIP {
			f, err := os.OpenFile(rpzZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}

			defer f.Close()

			if _, err = fmt.Fprintln(f, `;## Additional RPZ passthroughs`); err != nil {
				return err
			}

			revIP := reverseIPv4(strings.Split(ip, "."))
			if _, err = fmt.Fprintln(f, `32.`+revIP+`.rpz-client-ip      CNAME rpz-passthru.`); err != nil {
				return err
			}
		}
	}

	if _, err := os.Stat(customZone); os.IsNotExist(err) {
		f, err := os.Create(customZone)
		if err != nil {
			return err
		}

		defer f.Close()

		if _, err = fmt.Fprintln(f, ""); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(rpzZone, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = fmt.Fprintln(f, "$INCLUDE "+customZone); err != nil {
		return err
	}

	if dns := os.Getenv("UPSTREAM_DNS"); dns != "" {
		f, err := ioutil.ReadFile(namedConf)
		if err != nil {
			return err
		}

		lines := strings.Split(string(f), "\n")

		r := strings.NewReplacer("#ENABLE_UPSTREAM_DNS#", "", "dns_ip", dns)
		if dnssec := os.Getenv("ENABLE_DNSSEC_VALIDATION"); dnssec == "true" {
			r = strings.NewReplacer("#ENABLE_UPSTREAM_DNS#", "", "dns_ip", dns, "dnssec-validation no", "dnssec-validation auto")
		}

		for i, line := range lines {
			lines[i] = r.Replace(line)
		}

		output := strings.Join(lines, "\n")
		if err = ioutil.WriteFile(namedConf, []byte(output), 0644); err != nil {
			return err
		}
	}

	log.Print("Finished bootstrapping.")

	return nil
}
