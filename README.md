This is an Wi-Fi gateway for GreenBank G-Switch.

When it begins running, it will start an http server and listen on port `2304` by default, then scan the switches around.

Visit `http://{{ip}}:2304/status` and you will get a switch list like this:

```
{"24:71:89:0a:31:95":{"mac_address":"2471890a3195","light1":true,"light2":false,"light3":false,"pair_flag":false,"product_type":0,"DialAddress":"24:71:89:0a:31:95"}}
```

You can control your light by a command like this:

```
http://192.168.1.191:2304/control?light2=0&mac_address=2471890a3195
```



