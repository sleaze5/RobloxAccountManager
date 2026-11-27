# Writing

These rules apply to all text in the project: UI text, notifications, errors, logs, code comments, doc comments, commit messages, pull requests, issues, release notes, and documents. They are based on ASD-STE100 Simplified Technical English.

Write every text as simply and as shortly as possible. Remove words, not information. A short text that loses a fact that the reader needs is wrong.

Rules for one kind of text are in these files. They add to this file:

- UI text: Writing and Capitalization in [DESIGN.md](./DESIGN.md#writing)
- Errors: Errors in [ARCHITECTURE.md](./ARCHITECTURE.md#errors)
- Logs: Logging in [ARCHITECTURE.md](./ARCHITECTURE.md#logging)
- Commits and pull requests: [GIT.md](./GIT.md)

Before you finish a change, complete the [Writing checklist](#writing-checklist).

## Words

- Use only the terms in [GLOSSARY.md](./GLOSSARY.md) for the concepts that it names. Each term has one meaning, and each meaning has one term.
- Do not use the words in the "Do not use" column of the glossary for that meaning, even when Roblox uses them.
- When a text needs a concept that the glossary does not name, add the term to the glossary in the same change.
- Use common words. Write "use", not "utilize". Write "start", not "initiate". Write "help", not "facilitate".
- Use a word only as one part of speech. Do not use a noun as a verb, such as "to cookie an account".
- Use a verb to say what something does. Do not hide the verb in a noun: write "check the cookie", not "perform a check of the cookie".
- Do not use phrasal verbs, such as "set up", "carry out", or "find out", unless the glossary names them, such as "Turn on" and "Put on". Write "configure", "do", or "find".
- Do not use slang, idioms, jokes, or figures of speech.
- Do not use vague words, such as "stuff", "things", "various", "some", "handle", "deal with", or "etc.". Name the items.
- Keep a noun cluster to 3 words at most. Write "the expiry date of the cookie", not "the account cookie expiry date".

## Sentences

- Write one idea in each sentence.
- Keep an instruction to 20 words at most. Keep a description to 25 words at most.
- Use the active voice. Write "Roblox rejected the cookie", not "The cookie was rejected".
- Use the simple present, the simple past, or the simple future. Do not use "-ing" forms as verbs, except in progress labels such as "Saving…".
- Keep "the", "a", and "this". Do not remove articles to make a sentence shorter. Labels, headings, and button text are the only exception.
- Do not use contractions. Use US spelling.
- Write a number with digits and its unit, such as "30 seconds" or "5 MB".
- Do not use parentheses for a second idea. Write a second sentence.
- Do not use semicolons or em dashes. Write two sentences.

## Instructions

An instruction tells the reader to do something, such as a recovery step in an error or a step in a document.

- Start the instruction with a verb in the imperative, such as "Restart Roblox."
- Write one instruction in each sentence. Write two actions in one sentence only when the reader must do them at the same time.
- Put a condition before its instruction: "If the vault is locked, unlock it."
- Write steps that must happen in order as a numbered list.
- Put a warning before the step that it applies to. Start it with the command, then give the reason: "Do not close the app. The backup is not complete."

## Descriptions

A description tells the reader what something is or does, such as a setting description, a doc comment, or a pull request description.

- Put the most important information first.
- Write one topic in each paragraph, with at most 6 sentences.
- Use a list for 3 or more parallel items.
- Say what something does and why. Do not say how the code does it when the code already shows that.

## UI text

UI text must be clear to a user who does not know how the app works, and precise enough for a power user.

- Name the thing that the user sees, not how the app does it. Write "Could not join the server", not "The launch request returned an error".
- Keep the exact detail that a power user needs to act: the Roblox limit, the setting name, the ID, the file, or the code. Do not remove it to make the text simpler.
- Do not show internal terms: function names, endpoints, HTTP status text, error text from code, or the names of fields in an API.
- Use a technical term only when it is in the glossary or when the user needs it to act, such as `.ROBLOSECURITY`.
- Use the glossary term for each concept everywhere: labels, buttons, tooltips, notifications, errors, empty states, and screen reader labels.
- Give an action the same verb in every place: the button, the menu item, the dialog title, the progress label, the notification, and the error.

## Code comments

- Write a comment only for a fact that the code cannot show, as [AGENTS.md](../AGENTS.md) describes.
- Write full sentences that start with a capital letter and end with a period. A doc comment in Go starts with the name that it describes.
- Use the glossary terms for the concepts of the app. An identifier that copies a Roblox API field, such as `jobId` or `universeId`, keeps the API name. The comment uses the glossary term, such as "server ID".

## Logs

- Write log messages in lowercase, as [ARCHITECTURE.md](./ARCHITECTURE.md#logging) describes.
- Use the glossary terms. Write "could not join the server", not "could not join the instance".
- Put details in attributes, not in the message.

## Commits, pull requests, and issues

- Write the title as one short sentence in sentence case, without a final period. Start it with a verb in the imperative, such as "Rename groups in the UI".
- Name what changed for the user, not the files that changed.
- Use the glossary terms.
- Keep a pull request description to a short list of the changes and the reasons.

## Writing checklist

Check each text that the change adds or changes. If an item fails, rewrite the text.

Words:

- [ ] **Glossary:** Each concept uses its term from [GLOSSARY.md](./GLOSSARY.md). No word from the "Do not use" column has that meaning.
- [ ] **One meaning:** Each term has the meaning that the glossary gives it, and no other meaning.
- [ ] **New terms:** A concept that the glossary does not name has a new glossary entry in the same change.
- [ ] **Verbs:** The same action has the same verb in each place: button, menu item, dialog, progress label, notification, error, log, and comment.
- [ ] **Common words:** No complex word has a common alternative. No slang, idioms, vague words, or phrasal verbs that the glossary does not name.
- [ ] **Noun clusters:** No noun cluster has more than 3 words.

Sentences:

- [ ] **One idea:** Each sentence has one idea. Each instruction has one action.
- [ ] **Length:** Instructions have 20 words or fewer. Descriptions have 25 words or fewer.
- [ ] **Voice and tense:** Each sentence uses the active voice and a simple tense.
- [ ] **Articles:** Sentences keep "the", "a", and "this".
- [ ] **Order:** Conditions come before their instructions. The most important information comes first.
- [ ] **Format:** No contractions, parentheses for a second idea, semicolons, or em dashes. US spelling. Numbers as digits with units.

Content:

- [ ] **Short:** No word can be removed without loss of a fact or of clarity.
- [ ] **Complete:** Each fact that the reader needs to act is still there, such as an ID, a limit, a code, or a recovery step.
- [ ] **UI level:** UI text is clear to a new user and precise for a power user. It shows no internal terms.
- [ ] **Purpose:** A comment states a fact that the code cannot show.
- [ ] **Kind of text:** The text also follows the rules for its kind in [DESIGN.md](./DESIGN.md#writing), [ARCHITECTURE.md](./ARCHITECTURE.md), or [GIT.md](./GIT.md).
