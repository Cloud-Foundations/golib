# gitdb
A database for user and group information using Git as the back-end.

The gitdb package will periodically pull a specified remote Git repository to a
local directory. Whenever there is a new commit the local copy is scanned for
user and group information (i.e. group memberships). If the remote Git
repository becomes unavailable the local copy is used.

The database is read from `groups.json` files in directories in the repository.
All the groups files are merged together; the directory structure is not
relevant to how the repository is processed. This allows for arbitrary directory
structures to reflect the organisation. Each directory must have the following
files:
- `groups.json`: containing group definitions and their memberships
- `permitted-groups.json`: containing a list of regular expressions for the
                           permitted groups in the `groups.json` file

If a group is defined in the `groups.json` file but the group name does not
match one of the regular expressions in the `permitted-groups.json` file in the
same directory, that group definition is ignored. By using an access control
mechanism like
[GitHub CODEOWNERS](https://help.github.com/en/github/creating-cloning-and-archiving-repositories/about-code-owners)
it becomes possible to delegate control over `groups.json` files (i.e. delegate
control over team group memberships) while retaining central control over access
group memberships and the `permitted-groups.json` files.

An example is shown in the [example](example) directory.
- membership of Engineering groups has been delegated to alice and dave
- membership Finance has been delegated to gwen
- frank controls everything, including which groups are permitted to access AWS
  roles and the delegation rules.
