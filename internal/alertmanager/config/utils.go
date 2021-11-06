package config

import (
	"strconv"

	amcfg "github.com/prometheus/alertmanager/config"
)

func getReceiverPosition(conf *amcfg.Config, receiver string) int64 {
	for index, value := range conf.Receivers {
		if value.Name == receiver {
			return int64(index)
		}
	}

	return -1
}

// get given route position in config file
func getRoutePosition(routes []*amcfg.Route, receiver string, match map[string]string) int64 {
	for index, value := range routes {
		if value.Receiver == receiver {
			if match != nil {
				if value.Match["alertgroup"] == match["alertgroup"] {
					return int64(index)
				}
			} else {
				return int64(index)
			}
		}
	}

	return -1
}

func removeReceiver(conf *amcfg.Config, receiver int64) error {
	r := strconv.FormatInt(receiver, 10)
	if pos := getReceiverPosition(conf, r); pos != -1 {
		conf.Receivers[pos] = conf.Receivers[len(conf.Receivers)-1]
		conf.Receivers = conf.Receivers[:len(conf.Receivers)-1]
	}
	return nil
}

func delAllRoutes(conf *amcfg.Config, receiver int64) error {
	r := strconv.FormatInt(receiver, 10)

	var routes []*amcfg.Route
	for _, route := range conf.Route.Routes {
		if r != route.Receiver {
			routes = append(routes, route)
		}
	}

	conf.Route.Routes = routes

	return nil
}
