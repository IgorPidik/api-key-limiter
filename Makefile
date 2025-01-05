include .env

run:
	go run .

test:
	https_proxy=https://${PROXY_CONFIG_ID}:${PROXY_PROJECT_ID}:${PROXY_ACCESS_KEY}@localhost:9000 curl --proxy-insecure -v --cacert certs/ca.pem https://google.com 

