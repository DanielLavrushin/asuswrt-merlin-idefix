#!/bin/sh

wget -q --show-progress -O /tmp/asuswrt-merlin-idefix.tar.gz https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-idefix.tar.gz

tar -xzf /tmp/asuswrt-merlin-idefix.tar.gz -C /tmp

[ -d /jffs/.asdbk ] && rm -f /jffs/.asdbk/*idefix* 2>/dev/null

mkdir -p /jffs/scripts
cp -f /tmp/idefix/idefix /jffs/scripts/idefix
chmod 0755 /jffs/scripts/idefix /tmp/idefix/idefix

sh /tmp/idefix/idefix update

[ -d /jffs/.asdbk ] && rm -f /jffs/.asdbk/*idefix* 2>/dev/null
cp -f /tmp/idefix/idefix /jffs/scripts/idefix
chmod 0755 /jffs/scripts/idefix
