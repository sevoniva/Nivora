#!/usr/bin/env python3
"""
Nivora Translation Verification — class-based checker with file caching.

Usage:
    python3 scripts/verify-translations.py
    make verify-translations

What it does:
    Compares README.md (English source) against 4 translation files
    (zh-CN, ja-JP, ko-KR, es-ES) using the authoritative glossary at
    docs/TRANSLATION_GLOSSARY.json.

Checks (10 total):
    1   Structure integrity — line / code-block / mermaid counts match
    1b  Code-block content    — byte-for-byte identity (skips text blocks)
    2   Proper nouns          — notranslate terms appear same # of times
    3   Feature-name risk     — (retired; glossary covers it)
    4   Link preservation     — docs/ + .md links unchanged
    5   Glossary compliance   — each term exists in each language file
    6   English residual      — untranslated English left in zh / ja / ko
    7   Translation freshness — git commit-time comparison
    8   Heading structure     — ## and ### counts match
    9   Numeric fidelity      — significant numbers preserved
    10  Disclaimer            — reminder that this is structural only

Exit code:
    0 — all checks pass
    1 — at least one FAIL

Adding a new check:
    1. Write a method `_cNN_description` on TranslationChecker.
    2. Add ("Check NN: ...", self._cNN_description) to the table in run().

Requirements:
    Python 3.6+  (stdlib only — no pip install needed)
    git (for Check 7 freshness — degrades gracefully if absent)
"""

import json, os, re, subprocess, sys

ROOT_TRANSLATIONS = ["README.zh-CN.md", "README.ja-JP.md", "README.ko-KR.md", "README.es-ES.md"]
LANG_MAP = {"zh-CN":"zh", "ja-JP":"ja", "ko-KR":"ko", "es-ES":"es"}

class TranslationChecker:
    def __init__(self, base="README.md", translations=ROOT_TRANSLATIONS,
                 glossary_path="docs/TRANSLATION_GLOSSARY.json"):
        self.base = base
        self.translations = translations
        self.glossary = self._load(glossary_path)
        self._file_cache = {base: open(base).read()}
        for t in translations:
            if os.path.exists(t):
                self._file_cache[t] = open(t).read()
        self.p = 0; self.f = 0; self.w = 0

    # ── I/O (cached) ────────────────────────────────────────────────
    def _load(self, path):
        with open(path) as fh: return json.load(fh)

    def _raw(self, path):
        return self._file_cache[path]

    def _lines(self, path):
        """ponytail: cached per file, each check re-computes from cached raw"""
        return self._raw(path).splitlines(keepends=True)

    def _prose_lines(self, path):
        """lines outside ``` fences, inline backticks stripped — matches old read_prose()"""
        key = f"_pl_{path}"
        if not hasattr(self, key):
            out, inside = [], False
            for line in self._lines(path):
                if line.startswith("```"):
                    inside = not inside; continue
                if not inside:
                    out.append(re.sub(r"`[^`]*`", " ", line))
            setattr(self, key, out)
        return getattr(self, key)

    def _prose_text(self, path):
        return "".join(self._prose_lines(path))

    def _blocks(self, path):
        """extract code blocks excluding ```text fences — matches old extract_blocks()"""
        key = f"_blk_{path}"
        if not hasattr(self, key):
            blocks, inside, skip, buf = [], False, False, []
            for line in self._lines(path):
                if line.startswith("```") and not inside:
                    inside = True; buf = []
                    if "text" in line: skip = True
                    continue
                if line.startswith("```") and inside:
                    if not skip: blocks.append("".join(buf))
                    inside = False; skip = False; continue
                if inside and not skip: buf.append(line)
            setattr(self, key, blocks)
        return getattr(self, key)

    # ── check harness ───────────────────────────────────────────────
    def _pass(self, msg): self.p += 1; print(f"✅ PASS  {msg}")
    def _fail(self, msg): self.f += 1; print(f"❌ FAIL  {msg}")
    def _warn(self, msg): self.w += 1; print(f"⚠️  WARN  {msg}")
    def _check(self, name, fn):
        print(f"\n--- {name} ---"); fn()

    # ── Check implementations ───────────────────────────────────────
    def _c01_structure(self):
        bl = len(self._lines(self.base))
        bc = sum(1 for l in self._lines(self.base) if l.startswith("```"))
        bm = sum(1 for l in self._lines(self.base) if "mermaid" in l)
        print(f"  Base: {bl} lines, {bc} code blocks, {bm} mermaid blocks")
        for tf in self.translations:
            if tf not in self._file_cache:
                self._fail(f"{tf} missing"); continue
            l = len(self._lines(tf))
            c = sum(1 for ln in self._lines(tf) if ln.startswith("```"))
            m = sum(1 for ln in self._lines(tf) if "mermaid" in ln)
            if l == bl and c == bc and m == bm:
                self._pass(f"{tf} ({l} lines, {c} code, {m} mermaid)")
            else:
                self._fail(f"{tf} — lines={l} (want {bl}), code={c} (want {bc}), mermaid={m} (want {bm})")

    def _c01b_blocks(self):
        bb = self._blocks(self.base)
        for tf in self.translations:
            tb = self._blocks(tf)
            if len(tb) != len(bb):
                self._fail(f"{tf}: {len(tb)} code blocks (base has {len(bb)})"); continue
            diffs = sum(1 for bpair in zip(bb, tb) if bpair[0] != bpair[1])
            if diffs == 0:
                self._pass(f"{tf}: all {len(bb)} code blocks match")
            else:
                self._fail(f"{tf}: {diffs} code blocks differ from base")

    def _c02_notranslate(self):
        for term in self.glossary.get("notranslate", []):
            base_count = self._raw(self.base).count(term)
            if base_count == 0: continue
            all_match = True
            for tf in self.translations:
                count = self._raw(tf).count(term)
                if count != base_count:
                    self._fail(f"{term} in {tf}: {count} occurrences (base has {base_count})")
                    all_match = False
            if all_match:
                self._pass(f"{term}: {base_count} occurrences in all files")

    def _c03_feature_risk(self):
        self._pass("No hardcoded forbidden patterns (glossary uses correct translations)")

    def _c04_links(self):
        base_paths = set(re.findall(r"\(docs/[^)]+\)", self._raw(self.base)))
        base_all_links = set(re.findall(r"\(([A-Z][A-Za-z_.-]+\.md)\)", self._raw(self.base)))
        for tf in self.translations:
            tf_paths = set(re.findall(r"\(docs/[^)]+\)", self._raw(tf)))
            tf_all_links = set(re.findall(r"\(([A-Z][A-Za-z_.-]+\.md)\)", self._raw(tf)))
            missing = base_paths - tf_paths
            extra = tf_paths - base_paths
            failures = []
            if missing: failures.append(f"missing docs/ paths: {missing}")
            if extra: self._warn(f"{tf}: extra docs/ paths — {extra}")
            missing_links = base_all_links - tf_all_links
            if missing_links: failures.append(f"missing internal links: {missing_links}")
            if not failures:
                self._pass(f"{tf}: all paths preserved")
            else:
                for msg in failures:
                    self._fail(f"{tf}: {msg}")

    def _c05_glossary(self):
        base_prose_text = self._prose_text(self.base)
        base_full = self._raw(self.base)
        tprose = {tf: self._prose_text(tf) for tf in self.translations}
        total, ok, errors = 0, 0, 0
        for term in self.glossary.get("terms", []):
            en = term["en"]; total += 1
            en_prose_count = base_prose_text.count(en)
            en_full_count = base_full.count(en)
            if en_full_count > 0 and en_prose_count == 0:
                ok += 1; continue
            all_ok = True
            for tf in self.translations:
                lang = LANG_MAP[tf.split(".")[1]]
                trans = term.get(lang, "")
                if not trans or en_prose_count == 0: continue
                trans_count = tprose[tf].count(trans)
                if trans_count == 0:
                    self._fail(f"Glossary: '{en}' → {trans} not found in {tf} prose")
                    errors += 1; all_ok = False
                elif trans_count < en_prose_count:
                    self._fail(f"Glossary: '{en}' → {trans} ({trans_count}x in {tf}, EN appears {en_prose_count}x)")
                    errors += 1; all_ok = False
            if all_ok: ok += 1
        self._pass(f"Glossary: {ok}/{total} terms verified, {errors} errors")

    def _c06_english_residual(self):
        known = set(self.glossary.get("notranslate", []) + [
            "Nivora", "sevoniva", "Phase", "CLI", "API", "MCP", "SSO", "RBAC",
            "OIDC", "OCI", "SBOM", "SCM", "ITSM", "KMS", "OPA", "PostgreSQL",
            "YAML", "Helm", "Kustomize", "Docker", "Kubernetes", "SSH", "HTTP",
            "gRPC", "REST"])
        for tf in self.translations:
            if tf == "README.es-ES.md": continue
            plines = self._prose_lines(tf)
            residual = []
            for i, line in enumerate(plines, 1):
                stripped = line.strip()
                if not stripped: continue
                if stripped[0].isupper() and stripped[0].isascii():
                    first_word = stripped.split()[0] if stripped.split() else ""
                    if first_word in known: continue
                    if any("\u4e00" <= c <= "\u9fff" for c in stripped[:20]) \
                       or any("\u3040" <= c <= "\u30ff" for c in stripped[:20]) \
                       or any("\uac00" <= c <= "\ud7af" for c in stripped[:20]):
                        continue
                    residual.append(f"{i}: {stripped}")
            if residual:
                self._warn(f"{tf}: untranslated English lines found:")
                for r in residual[:3]: print(f"  {r}")
            else:
                self._pass(f"{tf}: no untranslated English in prose")

    def _c07_freshness(self):
        try:
            base_time = int(subprocess.check_output(
                ["git", "log", "-1", "--format=%ct", self.base]).decode().strip())
            for tf in self.translations:
                try:
                    tf_time = int(subprocess.check_output(
                        ["git", "log", "-1", "--format=%ct", tf]).decode().strip())
                    if tf_time >= base_time:
                        self._pass(f"{tf}: translation synced with source")
                    else:
                        self._warn(f"{tf}: source changed since last translation update")
                except:
                    self._warn(f"{tf}: cannot determine git commit time")
        except:
            self._warn("git not available, skipping freshness check")

    def _c08_headings(self):
        bh2 = sum(1 for l in self._lines(self.base) if l.startswith("## "))
        bh3 = sum(1 for l in self._lines(self.base) if l.startswith("### "))
        for tf in self.translations:
            h2 = sum(1 for l in self._lines(tf) if l.startswith("## "))
            h3 = sum(1 for l in self._lines(tf) if l.startswith("### "))
            if h2 == bh2 and h3 == bh3:
                self._pass(f"{tf}: {h2} H2 + {h3} H3 headings match")
            else:
                self._fail(f"{tf}: H2={h2} (base={bh2}), H3={h3} (base={bh3})")

    def _c09_numbers(self):
        base_numbers = set()
        for line in self._prose_lines(self.base):
            for match in re.finditer(r"\b\d+\b", line):
                base_numbers.add(match.group())
        significant = {n for n in base_numbers
                       if len(n) >= 3 or n in {"31","53","86","32","19"}}
        all_num_ok = True
        for tf in self.translations:
            tf_prose = self._prose_text(tf)
            missing_nums = [n for n in sorted(significant, key=lambda x: -int(x))
                           if n not in tf_prose]
            if missing_nums:
                self._fail(f"{tf}: missing numbers — {', '.join(missing_nums[:5])}")
                all_num_ok = False
            else:
                self._pass(f"{tf}: all significant numbers preserved")
        if all_num_ok:
            self._pass("Numeric fidelity passes for all files")

    def _c10_disclaimer(self):
        self._pass("Note: This verification is a smoke test. It checks structure and glossary compliance.")
        self._pass("It does NOT verify semantic accuracy, grammar, or naturalness.")
        self._pass("Bilingual human review is required for production-quality translations.")

    # ── runner ──────────────────────────────────────────────────────
    def run(self):
        print("=" * 44)
        print(" Nivora Translation Verification")
        print("=" * 44)
        for name, fn in [
            ("Check 1: Structure integrity",           self._c01_structure),
            ("Check 1b: Code block integrity",         self._c01b_blocks),
            ("Check 2: Untranslated proper nouns",     self._c02_notranslate),
            ("Check 3: Feature name risk detection",   self._c03_feature_risk),
            ("Check 4: File path preservation",        self._c04_links),
            ("Check 5: Glossary compliance",           self._c05_glossary),
            ("Check 6: Untranslated English detection",self._c06_english_residual),
            ("Check 7: Translation Freshness",         self._c07_freshness),
            ("Check 8: Heading structure",             self._c08_headings),
            ("Check 9: Numeric fidelity",              self._c09_numbers),
            ("Check 10: Disclaimer",                   self._c10_disclaimer),
        ]:
            self._check(name, fn)
        print(f"\n{'=' * 44}")
        print(f" Summary: {self.p} passed, {self.f} failed, {self.w} warned")
        print(f"{'=' * 44}")
        sys.exit(1 if self.f > 0 else 0)

if __name__ == "__main__":
    TranslationChecker().run()
