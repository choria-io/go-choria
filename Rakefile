task :default => [:test]

ENV['MCOLLECTIVE_CERTNAME'] = 'rip.mcollective'

desc "Run just tests no measurements"
task :test do
  sh "ginkgo -r -skipMeasurements ."
end

desc "Run tests including measure tests"
task :test_and_measure do
  sh "ginkgo -r ."
end


desc "Builds packages"
task :build do
  version = ENV["VERSION"] || "0.0.0"
  sha = `git rev-parse --short HEAD`.chomp
  build = ENV["BUILD"] || "foss"
  packages = (ENV["PACKAGES"] || "").split(",")
  packages = ["el6_32", "el6_64", "el7_64", "el8_64", "xenial_64"] if packages.empty?
  go_version = ENV["GOVERSION"] || "1.15"

  source = "/go/src/github.com/choria-io/go-choria"

  packages.each do |pkg|
    if pkg =~ /^windows/
      builder = "choria/packager:stretch-go%s" % [go_version]
    elsif pkg =~ /^(.+?)_(.+)$/
      builder = "choria/packager:%s-go%s" % [$1, go_version]
    else
      builder = "choria/packager:el7-go%s" % go_version
    end

    sh 'docker run --rm -v `pwd`:%s -e SOURCE_DIR=%s -e ARTIFACTS=%s -e SHA1="%s" -e BUILD="%s" -e VERSION="%s" -e PACKAGE=%s %s' % [
      source,
      source,
      source,
      sha,
      build,
      version,
      pkg,
      builder
    ]
  end
end

desc "Builds binaries"
task :build_binaries do
  version = ENV["VERSION"] || "0.0.0"
  sha = `git rev-parse --short HEAD`.chomp
  build = ENV["BUILD"] || "foss"

  source = "/go/src/github.com/choria-io/go-choria"

  sh 'docker run --rm  -v `pwd`:%s -e SOURCE_DIR=%s -e ARTIFACTS=%s -e SHA1="%s" -e BUILD="%s" -e VERSION="%s" -e BINARY_ONLY=1 choria/packager:el7-go1.15' % [
    source,
    source,
    source,
    sha,
    build,
    version
  ]
end
