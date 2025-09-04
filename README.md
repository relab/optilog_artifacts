### OptiLog Artifacts

#### Setup

Use the following command to setup password-less access from the controller node to all other nodes.

```sh
ssh -o UpdateHostKeys=yes -o PreferredAuthentications=publickey -o StrictHostKeyChecking=no bbchain$i echo "hello bbchain$i"
```


