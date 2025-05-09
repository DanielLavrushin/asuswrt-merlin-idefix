#!/bin/sh

wget -q --show-progress -O /tmp/asuswrt-merlin-idefix.tar.gz https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-idefix.tar.gz

tar -xzf /tmp/asuswrt-merlin-idefix.tar.gz -C /tmp

mv /tmp/idefix/idefix /jffs/scripts/idefix

chmod 0755 /jffs/scripts/idefix

sh /jffs/scripts/idefix update
