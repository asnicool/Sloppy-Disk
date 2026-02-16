import Fuse from 'fuse.js'

const items = [
    { album: 'Goodbye Yellow Brick Road', artist: 'Elton John' },
    { album: 'Goodbye Cruel World', artist: 'Elvis Costello' },
    { album: 'Hello Goodbye', artist: 'The Beatles' },
    { album: 'Good Bye', artist: 'Unknown' }
]

const fields = ['album', 'artist']

const runTest = (query, threshold) => {
    console.log(`\nTesting query: '${query}' with threshold: ${threshold}`)
    const options = {
        keys: fields,
        includeScore: true,
        threshold: threshold,
        ignoreLocation: true,
        findAllMatches: true,
        minMatchCharLength: 2
    }
    const fuse = new Fuse(items, options)
    const results = fuse.search(query)
    console.log(`Matched: ${results.length}`)
    results.forEach(r => {
        console.log(`- ${r.item.album} (Score: ${r.score.toFixed(4)})`)
    })
}

runTest('goodbie', 0.4)
runTest('goodbie', 0.5)
runTest('goodbie', 0.6)
runTest('Goodbie', 0.4)
runTest('goodbie crual', 0.4)
runTest('goodbie crual', 0.6)
runTest('eltn', 0.4)
