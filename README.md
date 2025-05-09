![GitHub Release](https://img.shields.io/github/v/release/daniellavrushin/asuswrt-merlin-idefix?logoColor=violet)
![GitHub Release Date](https://img.shields.io/github/release-date/daniellavrushin/asuswrt-merlin-idefix)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/daniellavrushin/asuswrt-merlin-idefix/latest)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/5afa683e2930418a9b13efac6537aad8)](https://app.codacy.com/gh/DanielLavrushin/asuswrt-merlin-idefix/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
![GitHub Downloads (latest)](https://img.shields.io/github/downloads/daniellavrushin/asuswrt-merlin-idefix/latest/total)
![GitHub Downloads (total)](https://img.shields.io/github/downloads/DanielLavrushin/asuswrt-merlin-idefix/total?label=total%20downloads)

# DEFIX Terminal for ASUSWRT-Merlin

![image](https://github.com/user-attachments/assets/d535a0da-0d06-44a1-ba3a-7a7c21e84a72)


A self-contained erminal that plugs straight into the *Merlin* Web UI.


## Quick install

```bash
# SSH into your router as admin
wget -O /tmp/idefix-install.sh \
  https://raw.githubusercontent.com/DanielLavrushin/asuswrt-merlin-idefix/refs/heads/main/install.sh \
  && chmod 0755 /tmp/idefix-install.sh \
  && /tmp/idefix-install.sh \
  && rm /tmp/idefix-install.sh
```

## Uninstall

```bash
idefix uninstall
```


![image](https://github.com/user-attachments/assets/36f13b3d-57c3-4a50-a8ce-4fccc3de6c3d)
