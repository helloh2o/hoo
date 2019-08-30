# HTTP
./hox -addr ":31280"
# Basic Authentication
./hox -addr ":31280" -auth "username:passwd"
# TLS 
./hox -c c.crt -k k.key <br>
If you use candy, you can use ./hox -host your.domain.name
# Speed limiter
./hox -max 1080 <br>
When the transmission data reaches 10MB, the limit will take effect and the maximum speed is 1080kb/s
