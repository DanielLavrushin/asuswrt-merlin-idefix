#!/bin/sh

wget -O /tmp/asuswrt-merlin-idefix.tar.gz https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-idefix.tar.gz

tar -xzf /tmp/asuswrt-merlin-idefix.tar.gz -C /tmp

rm -rf /jffs/addons/idefix

mv /tmp/idefix/idefix /jffs/scripts/idefix

chmod 0755 /jffs/scripts/idefix

mkdir -p /opt/share/idefix

mv /tmp/idefix/app.js /opt/share/idefix
mv /tmp/idefix/index.asp /opt/share/idefix

rm -rf /tmp/asuswrt-merlin-idefix.tar.gz
rm -rf /tmp/idefix

sh /jffs/scripts/idefix install
