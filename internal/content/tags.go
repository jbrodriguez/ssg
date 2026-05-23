package content

import "sort"

// TagCount is a tag with its post count, used by the topic cloud / tag index.
type TagCount struct {
	Name  string
	Count int
	Link  string
}

// TagsFromPosts returns all tags sorted by descending count (ties broken by name).
func TagsFromPosts(posts []*Post) []TagCount {
	counts := map[string]int{}
	for _, p := range posts {
		for _, t := range p.Tags {
			counts[t]++
		}
	}
	out := make([]TagCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, TagCount{Name: name, Count: count, Link: "/tag/" + name + "/"})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// PostsByTag groups posts by tag name.  The slice for each tag preserves
// the input ordering (assumed sorted by date desc).
func PostsByTag(posts []*Post) map[string][]*Post {
	by := map[string][]*Post{}
	for _, p := range posts {
		for _, t := range p.Tags {
			by[t] = append(by[t], p)
		}
	}
	return by
}

// SimilarPosts returns up to limit posts that share tags with target, ranked
// by tag-overlap count (descending) then date (descending).
func SimilarPosts(target *Post, byTag map[string][]*Post, limit int) []*Post {
	type scored struct {
		post  *Post
		score int
	}
	cand := map[string]*scored{}
	for _, t := range target.Tags {
		for _, p := range byTag[t] {
			if p.Slug == target.Slug {
				continue
			}
			if s, ok := cand[p.Slug]; ok {
				s.score++
			} else {
				cand[p.Slug] = &scored{post: p, score: 1}
			}
		}
	}
	out := make([]*scored, 0, len(cand))
	for _, s := range cand {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].post.Date.After(out[j].post.Date)
	})
	if limit > len(out) {
		limit = len(out)
	}
	result := make([]*Post, limit)
	for i := 0; i < limit; i++ {
		result[i] = out[i].post
	}
	return result
}
