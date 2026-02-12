package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/openrdap/rdap"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const env = "QUERY_DOMAINS"

var client = &rdap.Client{}
var current = map[string]*Info{}
var lock = sync.RWMutex{}

type Info struct {
	Reg       string
	Exp       string
	Registrar string

	Time time.Time
}

func main() {
	targetDomains := domains()

	if len(targetDomains) == 0 {
		log.Fatalf("No domain specified! Specify a comma-separated in the %s environment variable", env)
	}

	g := gin.Default()
	g.GET("/metrics", func(c *gin.Context) {
		for _, targetDomain := range targetDomains {
			c, has := current[targetDomain]
			if has {
				if c != nil && time.Now().Sub(c.Time) >= 30*time.Second {
					has = false
					lock.Lock()
					current[targetDomain] = nil
					lock.Unlock()
				}
			}

			if !has {
				lock.Lock()
				if current[targetDomain] == nil {
					res, err := resolve(targetDomain)
					if err != nil {
						log.Printf("Error resolving domain %s: %v", targetDomain, err)
					} else {
						current[targetDomain] = res
					}
				}
				lock.Unlock()
			}
		}
		lock.RLock()
		defer lock.RUnlock()
		c.Status(200)
		for domain, info := range current {
			reg, _ := time.Parse(time.RFC3339, info.Reg)
			exp, _ := time.Parse(time.RFC3339, info.Exp)
			_, _ = c.Writer.WriteString(fmt.Sprintf("domain_registered_time{domain=\"%s\"} %d\n", domain, reg.Unix()))
			_, _ = c.Writer.WriteString(fmt.Sprintf("domain_expiration_time{domain=\"%s\"} %d\n", domain, exp.Unix()))
			_, _ = c.Writer.WriteString(fmt.Sprintf("domain_registrar_info{domain=\"%s\", registrar=\"%s\"} 1\n", domain, info.Registrar))
		}
	})
	if err := g.Run(":8889"); err != nil {
		log.Fatal(err)
	}
}

func domains() []string {
	list := strings.Split(os.Getenv(env), ",")
	var result []string

	for _, domain := range list {
		trimmed := strings.TrimSpace(domain)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func resolve(name string) (*Info, error) {
	request := &rdap.Request{
		Type:  rdap.DomainRequest,
		Query: name,
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	domain, ok := response.Object.(*rdap.Domain)
	if !ok {
		return nil, fmt.Errorf("invalid response type: expected *rdap.Domain, got %T", response.Object)
	}

	res := Info{
		Time: time.Now(),
	}

	for _, evt := range domain.Events {
		switch evt.Action {
		case "registration":
			{
				res.Reg = evt.Date
				break
			}
		case "expiration":
			{
				res.Exp = evt.Date
				break
			}
		}
	}
	for _, entity := range domain.Entities {
		if entity.VCard == nil {
			continue
		}
		if len(entity.Roles) > 0 {
			switch entity.Roles[0] {
			case "registrar":
				{
					for _, prop := range entity.VCard.Properties {
						if prop.Name == "fn" {
							if value, ok := prop.Value.(string); ok {
								res.Registrar = value
							} else {
								log.Printf("error parsing registrar data for %s: expected string, got %T", name, prop.Value)
							}

						}
					}
				}
			}
		}
	}

	if res.Exp == "" || res.Reg == "" {
		return nil, fmt.Errorf("missing expiry or registration date")
	}

	return &res, nil
}
