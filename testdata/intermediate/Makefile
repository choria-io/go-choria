all:
	@rm -f ${PWD}/*.csr ${PWD}/*.pem
	cfssl genkey -initca root.json | cfssljson -bare ca
	cfssl genkey -initca intermediate.json | cfssljson -bare intermediate
	cfssl sign -profile ca-to-root -ca ca.pem -ca-key ca-key.pem -config config.json intermediate.csr | cfssljson -bare intermediate
	cfssl genkey csr.json | cfssljson -bare rip.mcollective
	cfssl gencsr -key rip.mcollective-key.pem csr.json  | cfssljson -bare rip.mcollective
	cfssl sign -ca intermediate.pem -ca-key intermediate-key.pem rip.mcollective.csr subject.json | cfssljson -bare rip.mcollective && openssl x509 -in rip.mcollective.pem -noout -text
	cat rip.mcollective.pem intermediate.pem > chain-rip.mcollective.pem
	openssl verify -CAfile ca.pem -untrusted chain-rip.mcollective.pem chain-rip.mcollective.pem
	cp ca.pem certs/ca.pem
	cp chain-rip.mcollective.pem certs/rip.mcollective.pem
	
second:
	# Make second cert chain to test caching
	cfssl gencsr -key rip.mcollective-key.pem csr.json  | cfssljson -bare second-rip.mcollective
	cfssl sign -ca intermediate.pem -ca-key intermediate-key.pem rip.mcollective.csr subject.json | cfssljson -bare second-rip.mcollective && openssl x509 -in second-rip.mcollective.pem -noout -text
	cat second-rip.mcollective.pem intermediate.pem > second-chain-rip.mcollective.pem
	openssl x509 -in second-rip.mcollective.pem -noout -text

ca_chain:
	# Make sure that CA side intermediate chains work
	cat ca.pem intermediate.pem > certs/ca_chain_ca.pem
	cat rip.mcollective.pem > certs/ca_chain_rip.mcollective.pem
	openssl verify -CAfile certs/ca_chain_ca.pem certs/ca_chain_rip.mcollective.pem

email:
	cfssl genkey email.json | cfssljson -bare email.rip.mcollective
	cfssl gencsr -key email.rip.mcollective-key.pem email.json  | cfssljson -bare email.rip.mcollective
	cfssl sign -ca intermediate.pem -ca-key intermediate-key.pem email.rip.mcollective.csr subject.json | cfssljson -bare email.rip.mcollective && openssl x509 -in email.rip.mcollective.pem -noout -text
	cat email.rip.mcollective.pem intermediate.pem > email-chain-rip.mcollective.pem
	openssl verify -CAfile ca.pem -untrusted email-chain-rip.mcollective.pem email-chain-rip.mcollective.pem
	cp email-chain-rip.mcollective.pem certs/email-chain-rip.mcollective.pem

deploy:
	cp ca.pem certs/ca.pem
	cp chain-rip.mcollective.pem certs/rip.mcollective.pem
	cp second-chain-rip.mcollective.pem certs/second.rip.mcollective.pem


clean:
	rm -f *.pem *.csr
