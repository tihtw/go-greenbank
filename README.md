
This project is Wi-Fi gateway code for GreenBank G-Switch.

It will open http server and listen on 2304 port by default, and scan the switchs around.

Visit http://{{ip}}:2304/status and you will got switch list like this:

```
{"24:71:89:0a:31:95":{"mac_address":"2471890a3195","light1":true,"light2":false,"light3":false,"pair_flag":false,"product_type":0,"DialAddress":"24:71:89:0a:31:95"}}
```

and you can control you light by the command like this:

```
http://192.168.1.191:2304/control?light2=0&mac_address=2471890a3195
```



