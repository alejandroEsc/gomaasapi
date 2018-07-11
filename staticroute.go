// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE File for details.

package gomaasapi

import (
	"github.com/juju/errors"
	"github.com/juju/schema"
	"github.com/juju/version"
)

// StaticRoute defines an explicit route that users have requested to be added
// for a given Subnet.
type staticRoute struct {
	ResourceURI string

	ID int
	// Source is the Subnet that should have the route configured. (Machines
	// inside Source should use GatewayIP to reach Destination addresses.)
	Source *subnet
	// Destination is the Subnet that a MachineInterface wants to send packets to. We
	// want to configure a route to that Subnet via GatewayIP.
	Destination *subnet
	// GatewayIP is the IPAddress to direct traffic to.
	GatewayIP string
	// Metric is the routing Metric that determines whether this route will
	// take precedence over similar routes (there may be a route for 10/8, but
	// also a more concrete route for 10.0/16 that should take precedence if it
	// applies.) Metric should be a non-negative integer.
	Metric int
}

func readStaticRoutes(controllerVersion version.Number, source interface{}) ([]*staticRoute, error) {
	checker := schema.List(schema.StringMap(schema.Any()))
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "static-route base schema check failed")
	}
	valid := coerced.([]interface{})

	var deserialisationVersion version.Number
	for v := range staticRouteDeserializationFuncs {
		if v.Compare(deserialisationVersion) > 0 && v.Compare(controllerVersion) <= 0 {
			deserialisationVersion = v
		}
	}
	if deserialisationVersion == version.Zero {
		return nil, errors.Errorf("no static-route read func for version %s", controllerVersion)
	}
	readFunc := staticRouteDeserializationFuncs[deserialisationVersion]
	return readStaticRouteList(valid, readFunc)
}

// readStaticRouteList expects the Values of the sourceList to be string maps.
func readStaticRouteList(sourceList []interface{}, readFunc staticRouteDeserializationFunc) ([]*staticRoute, error) {
	result := make([]*staticRoute, 0, len(sourceList))
	for i, value := range sourceList {
		source, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.Errorf("unexpected value for static-route %d, %T", i, value)
		}
		staticRoute, err := readFunc(source)
		if err != nil {
			return nil, errors.Annotatef(err, "static-route %d", i)
		}
		result = append(result, staticRoute)
	}
	return result, nil
}

type staticRouteDeserializationFunc func(map[string]interface{}) (*staticRoute, error)

var staticRouteDeserializationFuncs = map[version.Number]staticRouteDeserializationFunc{
	twoDotOh: staticRoute_2_0,
}

func staticRoute_2_0(source map[string]interface{}) (*staticRoute, error) {
	fields := schema.Fields{
		"resource_uri": schema.String(),
		"ID":           schema.ForceInt(),
		"Source":       schema.StringMap(schema.Any()),
		"Destination":  schema.StringMap(schema.Any()),
		"gateway_ip":   schema.String(),
		"Metric":       schema.ForceInt(),
	}
	checker := schema.FieldMap(fields, nil) // no defaults
	coerced, err := checker.Coerce(source, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "static-route 2.0 schema check failed")
	}
	valid := coerced.(map[string]interface{})
	// From here we know that the map returned from the schema coercion
	// contains fields of the right type.

	// readSubnetList takes a list of interfaces. We happen to have 2 Subnets
	// to parse, that are in different keys, but we might as well wrap them up
	// together and pass them in.
	subnets, err := readSubnetList([]interface{}{valid["Source"], valid["Destination"]}, subnet_2_0)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(subnets) != 2 {
		// how could we get here?
		return nil, errors.Errorf("Subnets somehow parsed into the wrong number of items (expected 2): %d", len(subnets))
	}

	result := &staticRoute{
		ResourceURI: valid["resource_uri"].(string),
		ID:          valid["ID"].(int),
		GatewayIP:   valid["gateway_ip"].(string),
		Metric:      valid["Metric"].(int),
		Source:      subnets[0],
		Destination: subnets[1],
	}
	return result, nil
}
