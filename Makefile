DOMAIN=v.democry.org

fe-install:
	$(MAKE) install -C ./fe/

build:
	go build -o ./go/democry ./go/ 

supervisor-install:
	ssh root@${DOMAIN} apt update && ssh root@${DOMAIN} apt install supervisor

supervisor-stop:
	ssh root@${DOMAIN} service supervisor stop 

supervisor-start:
	ssh root@${DOMAIN} service supervisor start

upload:
	ssh root@${DOMAIN} mkdir -p /opt/vid/ /opt/vid/fe /opt/vid/go
	scp  -r ./go/democry root@${DOMAIN}:/opt/vid/go
	scp  -r ./go/ssl root@${DOMAIN}:/opt/vid/go
	scp  -r ./fe/src root@${DOMAIN}:/opt/vid/fe
	scp ./supervisor/v.conf root@${DOMAIN}:/etc/supervisor/conf.d/


deploy: supervisor-stop upload supervisor-start
