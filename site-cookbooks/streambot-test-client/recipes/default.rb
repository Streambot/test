include_recipe "apt"

include_recipe "graphite"
include_recipe "statsd"
include_recipe "golang"
include_recipe "chef-golang"
include_recipe "golang::packages"

# Verify, that StatsD is running on the machine
service "statsd" do
	action :restart
end