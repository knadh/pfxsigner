module github.com/knadh/pfxsigner

go 1.12

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/unidoc/unipdf/v3 v3.3.0
	github.com/urfave/cli v1.22.2
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7 // indirect
	software.sslmate.com/src/go-pkcs12 v0.0.0-20190322163127-6e380ad96778
)

replace github.com/unidoc/unipdf/v3 => github.com/knadh/unipdf/v3 v3.3.1-clean
