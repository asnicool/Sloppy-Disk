
import Fuse from 'fuse.js';

const items = [
  { id: '1', album: 'Goodbye Yellow Brick Road', artist: 'Elton John', date: '1973', genre: 'Rock' },
  { id: '2', album: 'Goodbye Mr. Mackenzie', artist: 'Goodbye Mr. Mackenzie', date: '1989', genre: 'Rock' },
  { id: '3', album: 'Good', artist: 'Morphine', date: '1992', genre: 'Rock' }
];

const options = {
  keys: ['album', 'artist', 'date', 'genre'],
  includeScore: true,
  threshold: 0.7,
  distance: 1000,
  ignoreLocation: true,
  findAllMatches: true,
  minMatchCharLength: 1,
  useExtendedSearch: true
};

const fuse = new Fuse(items, options);
const query = 'goodbie';
const results = fuse.search(query);

console.log('Results length:', results.length);
results.forEach(r => {
  console.log('Match:', r.item.album, 'by', r.item.artist, 'score:', r.score);
});

const matchedItems = results.map(result => ({
  ...result.item,
  _relevance: 1 - result.score
}));

console.log('Mapped item[0]:', JSON.stringify(matchedItems[0], null, 2));
