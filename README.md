# Mattermost Elrond

This repository houses the open-source components of Mattermost Elrond service. With its name inspired by the famous [Council of Elrond](https://www.google.com/url?q=https://lotr.fandom.com/wiki/Council_of_Elrond&sa=D&source=docs&ust=1649667594506252&usg=AOvVaw0Qb2WIxfAFYZPKbkXNaPIo) from the famous LOTR books that defined the fate of the Ring, this service will be used to manage and support ring-based deployments in Mattermost Cloud.

## Other Resources

This repository houses the open-source components of Mattermost Elrond. Other resources are linked below:

- [Mattermost Cloud](https://github.com/mattermost/mattermost-cloud)

## Get Involved

- [Join the discussion on Mattermost Cloud](https://community.mattermost.com/core/channels/cloud)


## Developing

### Environment Setup

#### Required Software

The following is required to properly run the elrond server.

##### Note: when versions are specified, it is extremely important to follow the requirement. Newer versions will often not work as expected

1. Install [Go](https://golang.org/doc/install)
2. Install and run local [Mattermost Cloud](https://github.com/mattermost/mattermost-cloud) server


### Building

Simply run the following:

```bash
go install ./cmd/elrond
alias elrond='$HOME/go/bin/elrond'
```

### Running
Before running the server the first time you must set up the DB with:

```bash
$ elrond schema migrate
```

Run the server with:

```bash
elrond server
```
tip: if you want to use a remote running Mattermost Cloud server pass the `--provisioner-server` flag


#### Ring
The ring reflects a group of Installation Groups that have a similar release purpose. Therefore a ring can have many registered Installation Groups. A number of registered installation groups higher than 1 can help to achieve canary releases. 

In a different terminal/window, to create a ring:
```bash
elrond ring create --name <ring-name> --priority <ring-priority> --version <cloud-image-version> --image <cloud-image>
i.e.
elrond ring create --name ring-1 --priority 1 --version test-1234 --image mattermost/mattermost-enterprise-edition
```
tip: You can register a Mattermost Installation group in the ring creation step. You can run `elrond ring create --help` to see more configuration options. 

#### Installation Group
The installation group reflects a group of Mattermost installations. Each ring can have multiple registered installation groups and each installation group should reflect a real Mattermost Cloud (provisioner) installation group. 

In a different terminal/window, to register an installation group:

```bash
elrond ring installation-group register --installation-group-name "<installation-group-name>" --provisioner-group-id "<mattermost-cloud-installation-group-id>" --ring "<ring-id>" --soak-time "<installation-group-soak-time>"
i.e
elrond ring installation-group register --installation-group-name "ig-1" --provisioner-group-id "test12345" --ring "test123456" --soak-time 60
```

### Testing

Run the go tests to test:

```bash
$ go test ./...
```

### Deleting a ring and installation groups

To deregister an installation group from a ring first check the IGs registered:
```bash
elrond ring list
[
    {
        "ID": "123456789",
        "Name": "ring-1",
        "Priority": 2,
        "SoakTime": 60,
        "State": "stable",
        "Provisioner": "elrond",
        "ActiveReleaseID": "123456789",
        "DesiredReleaseID": "123456789",
        "CreateAt": 1659706964202,
        "DeleteAt": 0,
        "ReleaseAt": 1659954915507099000,
        "installationGroups": [
            {
                "id": "123456789",
                "name": "ig-1",
                "state": "stable",
                "releaseAt": 1659954825485290000,
                "soakTime": 60,
                "provisionerGroupID": "123456789",
                "LockAcquiredBy": null,
                "LockAcquiredAt": 0
            }
        ],
        "APISecurityLock": false,
        "LockAcquiredBy": null,
        "LockAcquiredAt": 0
    }
]
```
Get the ID and delete it:
```bash
elrond ring installation-group delete --installation-group "<installation-group-id" --ring "<ring-id>"
i.e.
elrond ring installation-group delete --installation-group "123456789" --ring "123456789"
```

To delete a ring:
```bash
elrond ring delete --ring "<ring-id>"
```


### Releasing a ring
To release a new Mattermost version to a ring you can use the following command
```bash
elrond ring release --image "<mattermost-image>" --version "<mattermost-image-version>" --ring "<ring-id>"
i.e.
elrond ring release --image mattermost/mattermost-enterprise-edition --version version-2 --ring "123456789"
```

If you want to relase to all rings with a single command you can run 
```bash
elrond ring release --image "<mattermost-image>" --version "<mattermost-image-version>" --all-rings
```

The Elrond will follow the priority numbers and release first the ring with the lowest priority number. Then after the soak time has passed it will move to the next ring based on priority. 

Elrond supports Mattermost Provisioner group environment variable changes. You can perform a ring release with same or new Mattermost version and change environment variables as well passing the flag as shown below:
```bash
elrond ring release --image "<mattermost-image>" --version "<mattermost-image-version>" --ring "<ring-id>" --env-variable "<ENV_VARIABLE_NAME:ENV_VARIABLE_VALUE>" --env-variable "<ENV_VARIABLE_NAME:ENV_VARIABLE_VALUE>"
i.e.
elrond ring release --image mattermost/mattermost-enterprise-edition --version version-2 --ring "123456789" --env-variable "MM_TEST:123"
```

### Forcing a ring release
There are cases that a force release is required for example for an urgent bug fix or security patch. When a force flag is passed the soak times are ignored and the release process will be a lot faster.

To force a release you can run 
```bash
elrond ring release --image "<mattermost-image>" --version "<mattermost-image-version>" --all-rings --force
```


