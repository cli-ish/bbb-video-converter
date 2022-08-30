# Golang port of tilmanmoser/bbb-video-download

Benefits of this port:
* Optimized for faster runtime. (650% faster when it's not in deskshare mode)
* No Node.js, so no troubles with node versions conflicts and packages

# Basic installation

```bash
# Install Docker
sudo apt-get update
sudo apt-get install ca-certificates curl gnupg lsb-release
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin
```

# How to Build and run

```bash
git clone https://github.com/cli-ish/bbb-video-converter.git
cd bbb-video-converter
docker build -t bbb-converter .
docker run -v /var/bigbluebutton/published/presentation/{internalid}:/recdir bbb-converter -i /recdir -o video.mp4
```