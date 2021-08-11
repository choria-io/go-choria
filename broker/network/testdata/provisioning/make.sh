choria tool jwt secure.jwt ../ssl/private_keys/rip.mcollective.pem --default --urls nats://example.net:4222 --token toomanysecrets
choria tool jwt insecure.jwt ../ssl/private_keys/rip.mcollective.pem --default --urls nats://example.net:4222 --token toomanysecrets --insecure
