// Benchmark mirror: lunr.js 2.3.9 in Node, same synthetic corpus as
// bench_test.go. Run: node bench_mirror.mjs
import lunr from 'lunr';

const words = [
  'search', 'index', 'query', 'document', 'ranking', 'score', 'vector',
  'token', 'pipeline', 'stemmer', 'language', 'client', 'server', 'data',
  'system', 'engine', 'library', 'performance', 'memory', 'build',
];
function benchDocs(n) {
  const docs = [];
  for (let i = 0; i < n; i++) {
    docs.push({
      id: String(i),
      title: `${words[i % 20]} ${words[(i * 7) % 20]} handbook`,
      body: `${words[(i * 3) % 20]} ${words[(i * 5) % 20]} ${words[(i * 11) % 20]} and ${words[(i * 13) % 20]}. ${words[(i * 17) % 20]} ${words[(i * 19) % 20]}, ${words[(i * 2) % 20]} ${words[(i * 4) % 20]} ${words[(i * 6) % 20]}.`,
    });
  }
  return docs;
}
function build(docs) {
  return lunr(function () {
    this.ref('id');
    this.field('title', { boost: 10 });
    this.field('body');
    docs.forEach((d) => this.add(d));
  });
}
function time(label, iters, fn) {
  fn(); fn(); fn();
  const start = process.hrtime.bigint();
  for (let i = 0; i < iters; i++) fn();
  const ms = Number(process.hrtime.bigint() - start) / 1e6 / iters;
  console.log(`${label}: ${ms.toFixed(2)} ms`);
}
const docs = benchDocs(2000);
time('build 2000 docs', 20, () => build(docs));
const idx = build(docs);
time('search hit ("search")', 2000, () => idx.search('search'));
time('search miss ("zzzqqx")', 2000, () => idx.search('zzzqqx'));
time('search fuzzy ("search~1")', 2000, () => idx.search('search~1'));
