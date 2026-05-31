package main

import (
	_ "github.com/kudig/kudig/pkg/analyzer/kernel"
	_ "github.com/kudig/kudig/pkg/analyzer/kubernetes"
	_ "github.com/kudig/kudig/pkg/analyzer/log"
	_ "github.com/kudig/kudig/pkg/analyzer/network"
	_ "github.com/kudig/kudig/pkg/analyzer/process"
	_ "github.com/kudig/kudig/pkg/analyzer/runtime"
	_ "github.com/kudig/kudig/pkg/analyzer/security"
	_ "github.com/kudig/kudig/pkg/analyzer/servicemesh"
	_ "github.com/kudig/kudig/pkg/analyzer/system"
)
