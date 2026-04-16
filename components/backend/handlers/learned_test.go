//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Learned Handler >", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {

	Describe("parseFrontmatter", func() {
		It("parses valid frontmatter with all fields", func() {
			content := "---\ntype: correction\ndate: 2026-04-01T14:30:00Z\ntitle: Use Pydantic v2\nsession: session-1\nproject: my-project\nauthor: Agent\n---\n\nAlways use Pydantic v2 BaseModel."
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["type"]).To(Equal("correction"))
			Expect(fm["date"]).To(Equal("2026-04-01T14:30:00Z"))
			Expect(fm["title"]).To(Equal("Use Pydantic v2"))
			Expect(fm["session"]).To(Equal("session-1"))
			Expect(fm["project"]).To(Equal("my-project"))
			Expect(fm["author"]).To(Equal("Agent"))
			Expect(body).To(Equal("Always use Pydantic v2 BaseModel."))
		})

		It("handles missing frontmatter", func() {
			content := "Just plain text."
			fm, body := parseFrontmatter(content)
			Expect(fm).To(BeNil())
			Expect(body).To(Equal("Just plain text."))
		})

		It("handles incomplete frontmatter delimiters", func() {
			content := "---\ntype: correction\nNo closing delimiter"
			fm, body := parseFrontmatter(content)
			Expect(fm).To(BeNil())
			Expect(body).To(Equal(content))
		})

		It("strips quotes from values", func() {
			content := "---\ntitle: \"Quoted Title\"\ntype: 'pattern'\ndate: 2026-04-01\n---\n\nBody."
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["title"]).To(Equal("Quoted Title"))
			Expect(fm["type"]).To(Equal("pattern"))
			Expect(body).To(Equal("Body."))
		})

		It("handles empty body", func() {
			content := "---\ntype: pattern\ntitle: Empty\ndate: 2026-04-01\n---\n"
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["type"]).To(Equal("pattern"))
			Expect(body).To(Equal(""))
		})

		It("handles optional fields being absent", func() {
			content := "---\ntype: pattern\ntitle: Minimal\ndate: 2026-04-01\n---\n\nBody text."
			fm, body := parseFrontmatter(content)
			Expect(fm).NotTo(BeNil())
			Expect(fm["type"]).To(Equal("pattern"))
			Expect(fm["title"]).To(Equal("Minimal"))
			Expect(fm["session"]).To(Equal(""))
			Expect(fm["project"]).To(Equal(""))
			Expect(fm["author"]).To(Equal(""))
			Expect(body).To(Equal("Body text."))
		})
	})

	Describe("collectMDPaths", func() {
		It("collects .md files from array", func() {
			input := []interface{}{
				map[string]interface{}{"name": "fix.md", "path": "docs/learned/corrections/fix.md", "type": "file"},
				map[string]interface{}{"name": "readme.txt", "path": "docs/learned/readme.txt", "type": "file"},
				map[string]interface{}{"name": "corrections", "path": "docs/learned/corrections", "type": "dir"},
			}
			paths := collectMDPaths(input)
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal("docs/learned/corrections/fix.md"))
		})

		It("returns empty for empty array", func() {
			paths := collectMDPaths([]interface{}{})
			Expect(paths).To(BeEmpty())
		})

		It("handles single file object", func() {
			input := map[string]interface{}{"name": "fix.md", "path": "docs/learned/fix.md", "type": "file"}
			paths := collectMDPaths(input)
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal("docs/learned/fix.md"))
		})

		It("skips directories", func() {
			input := []interface{}{
				map[string]interface{}{"name": "corrections", "path": "docs/learned/corrections", "type": "dir"},
				map[string]interface{}{"name": "patterns", "path": "docs/learned/patterns", "type": "dir"},
			}
			paths := collectMDPaths(input)
			Expect(paths).To(BeEmpty())
		})

		It("skips non-md files", func() {
			input := []interface{}{
				map[string]interface{}{"name": "image.png", "path": "docs/learned/image.png", "type": "file"},
				map[string]interface{}{"name": "notes.txt", "path": "docs/learned/notes.txt", "type": "file"},
			}
			paths := collectMDPaths(input)
			Expect(paths).To(BeEmpty())
		})

		It("handles case-insensitive .MD extension", func() {
			input := []interface{}{
				map[string]interface{}{"name": "FIX.MD", "path": "docs/learned/FIX.MD", "type": "file"},
			}
			paths := collectMDPaths(input)
			Expect(paths).To(HaveLen(1))
		})
	})
})
