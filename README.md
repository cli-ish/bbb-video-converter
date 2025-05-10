# Golang port of tilmanmoser/bbb-video-download

Benefits of this port:

* Optimized for faster runtime. (600% faster with 6 threads & when it's not in deskshare mode)
* No Node.js, so no troubles with node versions conflicts and packages

# Comparison

| Type                         | Node.js     | Go (1 Thread)    | Performance increase            |
|------------------------------|-------------|------------------|---------------------------------|
| Long recording no drawings   | 885s        | 892s             | ~ 0%                            |
| Big canvas not long duration | 43s         | 17s              | ~ 252%                          |
| 4h long with a lot of slides | 4455s       | 1851s            | ~ 240%                          |

The new go implementation is scalable over the -t tag which is not yet implemented in the node.js version.

With the same recording above "4h long with a lot of slides" we get the new times:

| Thread count | Time  | Speed increase (compared to node.js) |
|--------------|-------|--------------------------------------|
| 2            | 1115s | ~ 400%                               |
| 3            | 905s  | ~ 490%                               |
| 4            | 813s  | ~ 540%                               |
| 5            | 773s  | ~ 570%                               |
| 6            | 734s  | ~ 600%                               |

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

# Fetch with go install
```bash
# Install dependencies (ffmpeg) over apt/apk/pacman
go install github.com/cli-ish/bbb-video-converter@latest
bbb-video-converter -v
```