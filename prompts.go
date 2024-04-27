package nit

const reviewCommentsPrompt = `%s

The changes from the git diff:
%s

Review this pull request. Leave comments for specific positions in the diff when you have something constructive to say. Be critical as you have high standards. Don't point out the obvious. A good PR review is thoughtful and contructive with specific and actionable feedback on the changes.

Format your response like this:
Summary: Provide a concise summary of your comments on the pull request.
Event: APPROVE or COMMENT
1. File path: path/to/file.py
Position: 4
Comment: "Contructive comment..."
2. File path: path/to/anouther/file.py
Position: 16
Comment: "Anouther contructive comment ..."
... for as many comments as needed

The "Position" value is the number of lines down from the first "@@" hunk header in the file you want to add a comment. The line just below the "@@" line is position 1, the next line is position 2, and so on. The position in the diff continues to increase through lines of whitespace and additional hunks until the beginning of a new file which starts with "diff --git".
The "File path" for a comment is the path to the file as described on the line "diff --git a/path/to/file.py b/path/to/file.py". It should not start with a slash or the "a/" and "b/" prefixes that are used in the diff.
The "Event" value should be either "APPROVE" or "COMMENT". Use "APPROVE" when the changes are fine and can be me merged as is, even if you provide additional comments or suggestions. Use "COMMENT" when the changes are not ready to be merged yet and your feedback should be acted upon.

Begin!`

const reviewPostBodyPrompt = `Generate the request body to POST the pull request review notes to github. Format your response as a JSON object.

PR details:
%s

%s

Body Parameters
- *body*: string, Required
The body text of the pull request review. Put a summary of the changes here.
- *event*: string, Required
The review action you want to perform. The review actions include: APPROVE or COMMENT.
- *comments*: array of objects
    - "body": string, Required
    Text of the review comment.
    - "path": string, Required
    The relative path to the file that necessitates a comment. This should not start with a slash.
    - position: integer
    The position in the diff where you want to add a review comment.

Request body JSON:`

const commentReplyPrompt = `Write a response to this pull request comment: %s

Comment hunk:
%s

All comments in this thread. You are acting in this exchange as %s:
%s

If there is nothing to say, just return "noreply". Otherwise, be concise`
