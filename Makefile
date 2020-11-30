DOMAIN=v.democry.org

build:
	go build -o ./go/democry ./go/ 


push:
	ssh root@${DOMAIN} mkdir -p /opt/vid/
	scp  -r ./* root@${DOMAIN}:/opt/vid/
	scp ./start.sh root@${DOMAIN}:/root/
	ssh root@${DOMAIN} /root/start.sh 

