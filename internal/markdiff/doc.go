// Package markdiff captures test behavior as reviewable Markdown documents.
//
// A markdiff test builds a Markdown document that explains the system under
// test and records what it actually did. When the test finishes, the
// document is compared with a checked-in file. If the two differ, the test
// fails and prints a unified diff. Running the tests with an update flag
// rewrites the files instead, so a behavior change shows up in code review
// as a readable diff of prose, tables, and code blocks rather than as a
// pile of changed assertions.
//
// # Writing documents for reviewers
//
// The checked-in Markdown is the artifact reviewers read, often without
// opening the test. Write it for them:
//
//   - Give the document a title and a summary that says what component is
//     being exercised and why the captured behavior matters.
//   - Split the document into sections, one per behavior. Every section
//     requires a summary; use it to state the behavior in plain words
//     ("Pressing cw deletes to the end of the word and enters insert
//     mode").
//   - Before each captured block, use [Body.Text] to say what the block is
//     and what the reader should notice in it. A code block without
//     context forces the reader to reverse-engineer the test.
//   - Keep captured output deterministic. Strip timestamps, temporary
//     paths, and random IDs before recording them, or the document will
//     churn on every run.
//
// [New] and [Body.Section] fail the test when the summary is empty, so the
// minimum level of explanation is enforced by the API.
//
// # Example
//
//	func TestTokenizer(t *testing.T) {
//		doc := markdiff.New(t, "Tokenizer",
//			"The tokenizer splits prompt text into words for the vi "+
//				"motions. These cases pin down how it treats punctuation.")
//
//		sec := doc.Section("Punctuation",
//			"Punctuation is its own token, so `w` stops before commas.")
//		sec.Text("Input:")
//		sec.Code("text", "hello, world")
//		sec.Text("Tokens produced:")
//		sec.List(Tokenize("hello, world")...)
//	}
//
// # File layout
//
// Documents are written below testdata/markdiff in the package directory,
// with one file per test, named after the test. Subtests become nested
// directories, so TestTokenizer/punctuation is stored at
// testdata/markdiff/TestTokenizer/punctuation.md. Use [WithPath] to choose
// a different location, for example when one test produces several
// documents.
//
// # Updating
//
// Documents are compared automatically when the test (including its
// subtests) completes. To accept new output, run the tests with any of:
//
//	go test ./... -markdiff.update
//	go test ./... -update          # when the test binary defines -update
//	MARKDIFF_UPDATE=1 go test ./...
//
// Documents are not compared or written when the test has already failed,
// because a partially built document would only add noise.
//
// A [Doc] is not safe for concurrent use. Parallel subtests should each
// create their own document.
package markdiff
