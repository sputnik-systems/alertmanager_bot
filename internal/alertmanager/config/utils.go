package config

import (
	amcfg "github.com/prometheus/alertmanager/config"
)

type route struct {
	receiver string
	match    map[string]string
}

func listRoutes(in []*amcfg.Route, receiver string) []route {
	out := make([]route, 0)
	for _, value := range in {
		if value.Receiver == receiver {
			out = append(out, route{value.Receiver, value.Match})
		}
	}

	return out
}

func getReceiverPosition(receivers []*amcfg.Receiver, receiver string) int64 {
	for index, value := range receivers {
		if value.Name == receiver {
			return int64(index)
		}
	}

	return -1
}

// get given route position in config file
func getRoutePosition(in []*amcfg.Route, receiver string, match map[string]string) int64 {
	routes := listRoutes(in, receiver)
	for index, value := range routes {
		if match != nil {
			for k, v := range match {
				if vv, ok := value.match[k]; ok && v == vv {
					return int64(index)
				}
			}
		} else {
			if value.match == nil {
				return int64(index)
			}
		}
	}

	return -1
}

func removeAllRoutes(in []*amcfg.Route, receiver string) []*amcfg.Route {
	var out []*amcfg.Route

	for _, value := range in {
		if receiver != value.Receiver {
			out = append(out, value)
		}
	}

	return out
}
