#!/bin/sh

wget -q --show-progress -O /tmp/asuswrt-merlin-idefix.tar.gz https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-idefix.tar.gz

tar -xzf /tmp/asuswrt-merlin-idefix.tar.gz -C /tmp

# Drop any stale ASD quarantine copies from older releases that lived in /jffs/scripts.
[ -d /jffs/.asdbk ] && rm -f /jffs/.asdbk/*idefix* 2>/dev/null

chmod 0755 /tmp/idefix/idefix
sh /tmp/idefix/idefix update
