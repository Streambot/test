name              "streambot-test-client"
maintainer        "Martin Biermann"
maintainer_email  "info@martinbiermann.com"
license           "MIT"
description       "Configures a Streambot test client"
long_description  IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version           "0.0.1"

depends "apt"
depends "golang"
depends "chef-golang"
depends "statsd"
depends "graphite"