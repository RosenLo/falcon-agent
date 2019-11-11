package g

import (
	"log"

	"strings"

	"github.com/hashicorp/consul/api"
)

func GetConsulInfo() (svc, route, status string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recoverd in GetConsulInfo", r)
		}
	}()
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Println("Failed to new client, due to err: ", err)
		return
	}
	checks, err := client.Agent().Checks()
	if err != nil {
		log.Println("Failed to check the agent, due to err: ", err)
		return
	}
	for _, check := range checks {
		services, err := client.Agent().Services()
		if err != nil {
			log.Println("Failed to get the service, due to err: ", err)
			return
		}
		for _, src := range services {
			for _, tag := range src.Tags {
				if strings.Contains(tag, "route") {
					route = tag
					status = check.Status
					svc = check.ServiceID
					return
				}
			}
		}
	}
	return
}
