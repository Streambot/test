normal["go"] = {
	:version => "1.2",
	:scm => true,
	:binary => "/usr/local/go/bin/go",
	:gopath => "/opt/go",
	:packages => [
		"code.google.com/p/go-uuid/uuid",
		"github.com/op/go-logging",
		"github.com/jessevdk/go-flags",
		"github.com/mbiermann/go-cluster"
	]
}