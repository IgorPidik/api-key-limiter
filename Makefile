include .env

run:
	go run .

test:
	https_proxy=${PROXY_URL} curl --proxy-insecure -v --cacert certs/ca.pem https://google.com 

