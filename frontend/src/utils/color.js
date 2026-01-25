/**
 * Generates a consistent HSL color from a string.
 * Used for generating background colors for album cards based on artist/album/date.
 * 
 * @param {string} str - The input string to hash
 * @returns {string} - The CSS HSL color string
 */
export const generateHashColor = (str) => {
    if (!str) return 'hsl(220, 13%, 18%)'; // Default gray-800/40 equivalent
    
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    
    // Generate HSL values
    // Hue: 0-360, based on hash
    const h = Math.abs(hash % 360);
    
    // Saturation: 25-45% for a muted but colored look (pastel/elegant)
    const s = 25 + (Math.abs(hash) % 20); 
    
    // Lightness: 15-25% to keep it dark enough for white text, but visible
    const l = 15 + (Math.abs(hash) % 10);
    
    return `hsl(${h}, ${s}%, ${l}%)`;
};
