test:
	https_proxy=localhost:8000 curl -v https://example.org --cacert certs/ca.pem
test2:
	https_proxy=localhost:8000 curl -v https://google.com --cacert certs/ca.pem
