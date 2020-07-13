

build:
	go build -o ./go/democry ./go/ 


push:
	ssh root@local.democry.org mkdir -p /opt/vid/; \
	scp  -r ./* root@local.democry.org:/opt/vid/; \
	scp ./start.sh root@local.democry.org:/root/ ; \
	ssh root@local.democry.org /root/start.sh 

