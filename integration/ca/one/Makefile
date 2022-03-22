all:
	@rm -f ${PWD}/*.csr ${PWD}/*.pem
	cfssl genkey -initca root.json | cfssljson -bare ca
	cfssl genkey csr.json | cfssljson -bare rip.mcollective
	cfssl gencsr -key rip.mcollective-key.pem csr.json  | cfssljson -bare rip.mcollective
	cfssl sign -ca ca.pem -ca-key ca-key.pem rip.mcollective.csr subject.json | cfssljson -bare rip.mcollective && openssl x509 -in rip.mcollective.pem -noout -text
	openssl verify -CAfile ca.pem rip.mcollective.pem
	cp rip-mcollective.pem certs/
	cp ca.pem certs/ca.pem
	
localhost:
	cfssl genkey localhost.json | cfssljson -bare localhost
	cfssl gencsr -key localhost-key.pem localhost.json  | cfssljson -bare localhost
	cfssl sign -ca ca.pem -ca-key ca-key.pem localhost.csr localhost-subject.json | cfssljson -bare localhost
	openssl x509 -in localhost.pem -noout -text
	openssl verify -CAfile ca.pem localhost.pem
	cp localhost.pem certs/

clean:
	rm -f *.pem *.csr
