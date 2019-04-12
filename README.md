# jivosearch

## Ansible
Take host list from browser:
```
cat aws.json | jq -cr '.instances[].dnsName'
```
Start fleet:
```
cd ansible
ansible-playbook -i inventory.yaml -f 20 -e 'DB_PASS=XXXXXXXX' play.yaml
```
