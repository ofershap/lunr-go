// Golden generator for lunr-go differential tests.
// Run: node testdata/gen/gen.js  (requires lunr@2.3.9 pinned via package.json)
// Writes:
//   ../index_from_js.json  - docs, JS-serialized index, query results
//   ../verify_from_go.js output consumed separately
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import lunr from 'lunr';
const __dirname = path.dirname(fileURLToPath(import.meta.url));

const docs = [
  { id: 'a', title: 'Go in Practice', body: 'Building real applications with Go. Concurrency, channels, and testing.' },
  { id: 'b', title: 'The Go Programming Language', body: 'A thorough introduction to Go: types, methods, interfaces and goroutines.' },
  { id: 'c', title: 'JavaScript: The Good Parts', body: 'Crockford on the good parts of JavaScript: functions, objects, and inheritance.' },
  { id: 'd', title: 'Learning JavaScript Search', body: 'Client-side search with lunr.js, stemming, scoring and serialization. Search search search!' },
  { id: 'e', title: 'State-of-the-art full-text search', body: 'Well-known state-of-the-art techniques for re-indexing large-scale corpora.' },
  { id: 'f', title: 'Café menus and naïve users', body: 'Über café crème brûlée, naïve résumé coördinate Zürich.' },
  { id: 'g', title: 'Emoji docs 🚀 and astral planes', body: 'Rocket 🚀 emoji appear in real documents. Plane ✈️ emoji too.' },
  { id: 'h', title: '', body: '   ' },
  { id: '42', title: 'The Answer', body: 'The answer to life, the universe and everything: 42.' },
  { id: '7', title: 'Seven', body: 'Seven wonders. Seven seven seven.' },
  { id: '10', title: 'Ten', body: 'Ten is more than seven and two.' },
  { id: 'x1', title: 'running runner runs', body: 'Running runners run runs easily. Connection connections connected.' },
];

const idx = lunr(function () {
  this.ref('id');
  this.field('title', { boost: 10 });
  this.field('body');
  this.metadataWhitelist = ['position'];
  docs.forEach(function (doc) { this.add(doc); }, this);
});

const serialized = JSON.stringify(idx);

const queries = [
  'go',
  'javascript search',
  'title:go',
  'body:search^5',
  'run~1',
  'searh~2',
  'search*',
  '*ing',
  '*search*',
  '+search +lunr',
  '-javascript search',
  'the',
  'nonexistent',
  '42',
  'café',
  'CAFÉ',
  '🚀',
  'state-of-the-art',
  'state -of-the-art',
  'title:go body:concurrency',
  'go^10 javascript',
  'connect*',
  'runner^2 runs',
  '+title:seven',
  '\\~',
  'foo\\~2',
  'unknownfield:go',
  'foo^bar',
  'foo~bar',
  'title:',
  '',
  ' ',
  '+',
  '-',
  'title:go~1^3',
  'go title:javascript',
  'GO GO',
  '42^2',
  '~2',
  '^5',
];

// Seeded RNG for differential query-parser fixtures.
function mulberry32(a) {
  return function () {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const rng = mulberry32(20261114);
const atoms = ['go', 'javascript', 'search', 'café', '🚀', 'title:', 'body:', '^', '~', '~2', '^10', '+', '-', '*', 'ing*', '*ing', '\\:', '\\~', 'unknown:', '42', 'x', ' ', '\\', 'foo\\'];
const parserFixtures = [];
for (let i = 0; i < 250; i++) {
  const n = 1 + Math.floor(rng() * 5);
  let q = '';
  for (let j = 0; j < n; j++) q += (j && rng() < 0.7 ? ' ' : '') + atoms[Math.floor(rng() * atoms.length)];
  const query = new lunr.Query(['title', 'body']);
  let entry = { query: q };
  try {
    new lunr.QueryParser(q, query).parse();
    entry.clauses = query.clauses.map(function (c) {
      return {
        term: c.term, fields: c.fields, boost: c.boost,
        editDistance: 'editDistance' in c ? c.editDistance : null,
        usePipeline: c.usePipeline, presence: c.presence,
      };
    });
  } catch (e) {
    entry.error = String(e.message || e);
  }
  parserFixtures.push(entry);
}

const results = queries.map(function (q) {
  try {
    return {
      query: q,
      results: idx.search(q).map(function (r) {
        return { ref: r.ref, score: r.score, matchData: r.matchData.metadata };
      }),
    };
  } catch (e) {
    return { query: q, error: String(e.message || e) };
  }
});

// Stemmer differential: stem every distinct corpus token plus edge cases.
const stemmerWords = new Set(['','a','ab','y','yy','by','byye','yls','eyes','yes','oy','yoy','buy','enjoy','yell','sses','ies','eed','ational','ization','izing','triplicate','hopeful','goodness','revival','allowance','inference','airliner','gyroscopic','adjustable','defensible','irritant','replacement','adjustment','dependent','homologous','communism','activate','angulariti','effective','bowdlerize','probate','rate','cease','controll','roll']);
docs.forEach(function (d) {
  lunr.tokenizer(d.title + ' ' + d.body).forEach(function (t) { stemmerWords.add(t.toString()); });
});
const stems = {};
stemmerWords.forEach(function (w) { stems[w] = lunr.stemmer(new lunr.Token(w, {})).toString(); });

const out = { docs: docs, serialized: serialized, results: results, parser: parserFixtures, stems: stems };
fs.writeFileSync(path.join(__dirname, '..', 'index_from_js.json'), JSON.stringify(out, null, 1));
console.log('wrote index_from_js.json: serialized index ' + serialized.length + ' bytes, ' + results.length + ' queries');

// Interchange: if index_from_go.json exists (produced by the Go test), load
// it with lunr.js and check results match the JS-built index exactly.
const goPath = path.join(__dirname, '..', 'index_from_go.json');
if (fs.existsSync(goPath)) {
  const goSerialized = JSON.parse(fs.readFileSync(goPath, 'utf8')).serialized;
  const goIdx = lunr.Index.load(JSON.parse(goSerialized));
  let mismatch = 0;
  queries.forEach(function (q) {
    let a, b;
    try { a = JSON.stringify(idx.search(q).map(r => [r.ref, r.score])); } catch (e) { a = 'ERR:' + (e.message || e); }
    try { b = JSON.stringify(goIdx.search(q).map(r => [r.ref, r.score])); } catch (e) { b = 'ERR:' + (e.message || e); }
    if (a !== b) { mismatch++; console.log('MISMATCH ' + JSON.stringify(q) + '\n  js-idx: ' + a + '\n  go-idx: ' + b); }
  });
  console.log(mismatch === 0 ? 'GO->JS interchange: all queries match' : mismatch + ' mismatches');
}
