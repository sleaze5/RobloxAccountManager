# Glossary

Use these terms in all text, as [WRITING.md](./WRITING.md) describes. Each term has one meaning, and each meaning has one term.

- Use the term in the "Term" column for the meaning in the "Meaning" column. Do not use it for another meaning.
- Do not use a word in the "Do not use" column for that meaning, even when Roblox uses it. Roblox renames its features often. These terms stay the same.
- Write a term in sentence case. Capitalize it only at the start of a label or sentence, as in "Friends" and "Remove friend". Keep the capitals of acronyms, as in "Place ID".
- In code, name every identifier with the glossary term, such as `serverID`, also when a Roblox API uses another name. Where the code reads or writes the Roblox name, such as in a JSON tag, a URL parameter, or a Roblox value, add a comment that names both, such as "Roblox calls the server ID jobId." The "Roblox API" notes in the tables give these names.
- When a text needs a concept that this file does not name, add a row in the same change.

## App and vault

| Term            | Meaning                                                                 | Do not use                              |
| --------------- | ----------------------------------------------------------------------- | --------------------------------------- |
| app             | Roblox Account Manager.                                                 | application (in UI text), program, tool |
| vault           | The encrypted, password-protected store of the accounts and their data. | database (in UI text), storage          |
| master password | The password that unlocks the vault.                                    | vault password, passphrase              |
| account         | A Roblox account that is saved in the vault.                            | alt, profile, login                     |
| saved account   | An account, when the text must separate it from other users.            | stored account, managed account         |
| cookie          | The `.ROBLOSECURITY` cookie that signs in an account.                   | token, session, auth cookie             |
| tag             | A label that the user gives to accounts to sort and filter them.        | label, category                         |

## Users

| Term          | Meaning                                                                | Do not use                          |
| ------------- | ---------------------------------------------------------------------- | ----------------------------------- |
| user          | Any Roblox account, outside a server.                                  | person (in labels), member, profile |
| player        | A user in a server.                                                    | user (in a server)                  |
| display name  | The name that a user selects and that Roblox shows above the username. | nickname                            |
| username      | The unique name of a user, written after "@".                          | handle, login name                  |
| online status | Whether a user is offline, online, in a game, or in Studio.            | presence                            |
| blocked user  | A user that the account blocked on Roblox.                             | ignored user                        |

## Social

| Term                   | Meaning                                                                                                                                                      | Do not use                                        |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| connection             | Any social relationship between the account and a user, outgoing or incoming: friend, trusted friend, follower, following, or mutual. It never means friend. | friend (for all kinds)                            |
| friend                 | A user who has a friend relationship with the account on Roblox. The list is "Friends".                                                                      | connection (for a friend)                         |
| trusted friend         | A friend that the account and the user both marked as trusted. The list is "Trusted friends".                                                                | trusted connection                                |
| follower               | A user who follows the account. The list is "Followers".                                                                                                     | fan, subscriber                                   |
| following              | A user that the account follows. The list is "Following".                                                                                                    | followed user, followings                         |
| mutual                 | A user who follows the account and whom the account follows. The app infers it from Followers and Following. The list and the badge are "Mutuals".           | mutual follower, mutual friend, mutual connection |
| friends in common      | Friends that the account and a user both have.                                                                                                               | mutual friends                                    |
| friend request         | A request to become friends.                                                                                                                                 | connection request, invite                        |
| trusted friend request | A request from a friend to become trusted friends.                                                                                                           | trusted connection request, trusted request       |
| incoming               | Sent by a user to the account, such as an incoming friend request.                                                                                           | received, inbound                                 |
| outgoing               | Sent by the account to a user, such as an outgoing friend request.                                                                                           | sent (as an adjective), outbound, pending         |

## Games

| Term           | Meaning                                                                                                                                               | Do not use                                 |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| game           | A universe with all of its places, as users see it on Roblox.                                                                                         | experience, universe (in UI text)          |
| place          | One place of a game: the root place or a subplace.                                                                                                    | level, map, world                          |
| root place     | The place that Roblox opens when a user joins the game.                                                                                               | start place, main place                    |
| subplace       | A place of a game that is not the root place.                                                                                                         | sub-place, child place                     |
| universe       | The Roblox structure that holds the places of a game. Use it only in "Universe ID" and in comments about the Roblox API. In other text, write "game". | game (in comments about the API structure) |
| server         | One running copy of a place that players join.                                                                                                        | instance, job, session                     |
| private server | A server that only the users that its owner invites can join.                                                                                         | VIP server                                 |
| Place ID       | The unique number of a place. Roblox API: `placeId`.                                                                                                  | game ID                                    |
| Server ID      | The unique ID of a server. Roblox API: `jobId` or `gameId`.                                                                                           | job ID, instance ID, game ID               |
| Universe ID    | The unique number of the universe of a game. Roblox API: `universeId`.                                                                                | game ID, experience ID                     |

## Groups

| Term         | Meaning                                   | Do not use            |
| ------------ | ----------------------------------------- | --------------------- |
| group        | A Roblox group.                           | community             |
| member       | A user in a group.                        | user (in a group)     |
| role         | The rank of a member in a group.          | rank (in UI text)     |
| owner        | The user who owns a group, game, or item. | creator (for a group) |
| join request | A request from a user to join a group.    | membership request    |
| group chat   | A Roblox chat with more than 2 users.     | group (for a chat)    |

## Roblox programs

| Term             | Meaning                                                                                     | Do not use                                     |
| ---------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Roblox           | The Roblox Player, or the company and its service. The context makes the meaning clear.     | the game client, RobloxPlayerBeta (in UI text) |
| client           | One running copy of the Roblox Player that an account uses.                                 | instance, process (for a client), window       |
| Multi-client     | The setting that lets two or more clients run at the same time.                             | multi-instance, multiple clients               |
| process          | An operating system process, such as a Roblox Player process. Use it only for the OS level. | client (for an OS process), task               |
| Roblox Studio    | The Roblox editor. Write "Studio" after the first use, as in the online status "In Studio". | the editor                                     |
| instance         | A Roblox engine object in a place or a model file, such as a `Part`, as Explorer shows it.  | object, node (for an engine object)            |

## Actions

Use one verb for each action. Use it on the button, the menu item, the dialog, the progress label, the notification, the error, the log, and the comment.

| Verb                 | Meaning                                                                              | Do not use                            |
| -------------------- | ------------------------------------------------------------------------------------ | ------------------------------------- |
| Add                  | Put an item into the app or a list, such as an account or a tag.                     | create, insert, new                   |
| Add friend           | Send a friend request to a user.                                                     | send request, connect                 |
| Add trusted friend   | Send a trusted friend request to a friend.                                           | trust                                 |
| Accept and Decline   | Answer an incoming request.                                                          | approve, deny, reject, ignore         |
| Cancel               | Stop an action before it completes, or withdraw an outgoing request.                 | abort, revoke, withdraw               |
| Remove               | Take an account, a browser, or another item out of the app, or end a friendship.     | unfriend, delete (for an account)     |
| Delete               | Permanently erase data, such as a tag or a file.                                     | erase, destroy, remove (for data)     |
| Follow and Unfollow  | Start or stop following a user.                                                      | subscribe, unsubscribe                |
| Block and Unblock    | Start or stop blocking a user on Roblox.                                             | ignore, mute                          |
| Launch               | Start Roblox with one or more accounts.                                              | start, run, open (for Roblox)         |
| Join                 | Become a player in a server, or a member of a group.                                 | enter, connect                        |
| Close                | End Roblox processes, or close a window or dialog.                                   | kill, terminate, exit                 |
| Clear                | Empty a field, a search, a filter, or a selection.                                   | reset (for input), delete (for input) |
| Lock and Unlock      | Change the vault state.                                                              | open (for the vault), sign out        |
| Select               | Mark an option, an item, or a user.                                                  | choose, pick                          |
| Check                | Test something with Roblox or with a rule that the user sees, such as a cookie, a password, or a name with the Roblox text filter. | validate, verify                      |
| validate             | In code only: test input from the UI or a file against format rules before use. Never use it in UI text. | check (for input rules in code)       |
| Show and Hide        | Make something visible or not visible.                                               | reveal, display, toggle               |
| Refresh              | Load data again.                                                                     | reload, sync, update (for data)       |
| Turn on and Turn off | Change a switch. The states are "On" and "Off".                                      | enable, disable, toggle               |
| Dismiss              | Hide a notification or a message.                                                    | close (for a notification)            |
| Put on and Take off  | Change what the avatar wears. "Worn" names the state.                                | equip, unequip, wear                  |
| Update cookie        | Give a saved account a new cookie.                                                   | replace cookie, refresh cookie        |
| Copy                 | Put a value on the clipboard.                                                        | clone (for the clipboard)             |
| Open in browser      | Open the page of a user, a game, a group, or an item on the Roblox website.          | view on Roblox, visit                 |

Some words name a state that Roblox sets. They do not name an action of the app:

- "Verified" names only a status that Roblox gives, such as a verified email or a verified badge.
- "Disabled" names only a state that Roblox sets, such as a disabled account.
