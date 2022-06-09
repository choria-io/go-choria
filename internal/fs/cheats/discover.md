# discovery concepts documentation https://choria.io/docs/concepts/discovery/

# find nodes with a tagged as webservers by the configuration management system
choria find -C roles::apache

# shows matching nodes as a JSON document
choria find -C roles::apache --json

# finds nodes running a specific version of choria
choria find -S 'choria().version=="0.99.0.20220609"'
