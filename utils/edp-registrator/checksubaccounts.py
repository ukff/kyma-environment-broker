import json

# This script reads EDP registration data (edp-subaccounts.json) and compares it with SKRs stored in a file with format (skr-subaccounts.txt):
# Subaccount                             InstanceID                             Plan         state          createdat
# You can use the following command to get those rows:
# ./kcp rt -o custom=Subaccount:subAccountID,InstanceID:instanceID,Plan:servicePlanName,state:status.state,createdat:status.createdAt > skr-subaccounts.txt

f = open("edp-subaccounts.json")
data = json.load(f)

edpSubaccounts = dict()

for item in data:
    # print(item['name'])
    sbAcc = item['name']
    edpSubaccounts[sbAcc] = True

f.close()
count = 0
f = open("skr-subaccounts.txt")
notExistingSb = []
for item in f.readlines():
    item = item.strip(" \n")
    id = item.split(" ")[0]
    if id not in edpSubaccounts:
        print(item)
        count = count + 1
        notExistingSb.append("'"+id+"'")
f.close()
print(count)
print(", ".join(notExistingSb))
