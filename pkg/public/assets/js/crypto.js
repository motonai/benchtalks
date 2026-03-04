// random key generation
//
// gives out 32 random bytes = 256 (supposed to be secure) as key.
//
// Why:
// https://www.ssldragon.com/blog/256-bit-encryption/,
// https://www.clickssl.net/blog/256-bit-encryption,
// https://www.newsoftwares.net/blog/why-256-bit-isnt-always-stronger-than-128-bit/,
// https://en.wikipedia.org/wiki/Advanced_Encryption_Standard
//
function generateEncryptionKey() {
    return nacl.randomBytes(32);
}

// The keygen will be used for everything that has to be encrypted (urls,
// messages, timestamps, etc.)

// generates a random hex room ID
// length is in bytes, so 16 bytes = 32 hex chars
// uses nacl.randomBytes so it's cryptographically secure (same source as the
// key)
//
function generateId(length) {
    const bytes = nacl.randomBytes(length);
    let hex = '';
    for (let i = 0; i < bytes.length; i++) {
        hex += bytes[i].toString(16).padStart(2, '0');
    }
    return hex;
}

// https://en.wikipedia.org/wiki/Base64
function keyToString(key) {
    return nacl.util.encodeBase64(key);
}

// https://stackoverflow.com/questions/38718202/how-to-use-uint8array-uint16array-uin32array
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Uint8Array
//
function stringToKey(keyString){

    console.log('stringToKey input:', keyString);
    console.log('stringToKey input length:', keyString ? keyString.length : 0);

    try{
        const decoded = nacl.util.decodeBase64(keyString);
        //debuging
        console.log('stringToKey decoded:', decoded);
        console.log('stringToKey decoded length:', decoded ? decoded.length : 0);
        //the decodeBase64 makes base64 strings to bytes
        return decoded;
    } catch(error) {
        console.error('stringToKey ERROR:', error);
        console.error('Invalid encryption key format');
        return null;
    }
}

// messages
//
// treat messages as objects and turn them into json strings.
// Then turn the json strings to bytes via Uint8Array
function encryptMessage(messageObj,key){
    const messageJSON = JSON.stringify(messageObj);
    const messageBytes = nacl.util.decodeUTF8(messageJSON);


    // Generation of random series of numbers that are going to be used only
    // once. These are called nonces
    const nonce = nacl.randomBytes(24); //24 bytes for XSalsa20

    // message encryption
    // The nacl.secretbox does symmetric encryption with authentication and it
    // returns Uint8Array of encrypted data
    const encrypted = nacl.secretbox(messageBytes, nonce, key);

    // Combining the nonce + the encrypted data in order to send the nonce so
    // the receiver can decrypt
    const combined = new Uint8Array(nonce.length + encrypted.length);
    combined.set(nonce);
    combined.set(encrypted, nonce.length);

    // conversion to base64 to transmit
    return nacl.util.encodeBase64(combined);
}

// message decryption

// encryptedBase64 as string from server and Uint8Array(32) as key.
// If the decryption fails the message object is null
function decryptMessage(encryptedBase64, key) {
    try {
        const combined = nacl.util.decodeBase64(encryptedBase64);

        // spliting the message object
        const nonce = combined.slice(0, 24);
        const encrypted = combined.slice(24);

        // decrypting
        const decrypted = nacl.secretbox.open(encrypted, nonce, key);

        if (!decrypted) {
            // failure
            console.error('Decryption failed');
            return null;
        }

        const messageJSON = nacl.util.encodeUTF8(decrypted);

        return JSON.parse(messageJSON);
    }catch (error) {
        console.error('Error decrypting message:', error);
        return null;
    }
}

// encryption & decryption for images
//
// This turns the image into binary then u8a, get a nonce and encrypt the bytes
// and puts them together and converts that in base64.
async function encryptImage(file, key) {
    const arrayBuffer = await file.arrayBuffer();
    const imageBytes = new Uint8Array(arrayBuffer);
    const nonce = nacl.randomBytes(24);
    const encrypted = nacl.secretbox(imageBytes, nonce, key);
    const combined = new Uint8Array(nonce.length + encrypted.length);
    combined.set(nonce);
    combined.set(encrypted,nonce.length);
    return nacl.util.encodeBase64(combined);
}

// So this one takes the encryptedBase64 with the u8a as key and puts out what
// is called a "blob"

function decryptImage(encryptedBase64, key, mimeType) {
    try {
        const combined = nacl.util.decodeBase64(encryptedBase64);
        const nonce = combined.slice(0,24);
        const encrypted = combined.slice(24);
        const decrypted = nacl.secretbox.open(encrypted, nonce, key);

        if  (!decrypted) {
            //failure - same concept
            console.error('Image decryption failed');
            return null;
        }
        return new Blob([decrypted],{type: mimeType});
    } catch(error){
        console.error('Error decrypting image:',error);
        return null;
    }
}

// same as decryptImage but used when receiving images via WebSocket
// instead of downloading from server — the encrypted blob arrives
// directly in the message payload so we decrypt it straight away.
// mimeType defaults to image/jpeg if not known — the browser
// will usually figure out the real type from the bytes anyway.
function decryptImageFromBase64(encryptedBase64, key, mimeType = 'image/jpeg') {
    return decryptImage(encryptedBase64, key, mimeType);
}

// admin tokens

// generation random tokens. it will show on urls but as a base64 string
function generateAdminToken() {
    const tokenBytes = nacl.randomBytes(32);
    return nacl.util.encodeBase64(tokenBytes);
}

// hashing admin token BEFORE sending it to the server
async function hashAdminToken(tokenString) {
    const tokenBytes = nacl.util.decodeBase64(tokenString);

    // there's built-in crypto in browsers :o It's going to be used for hashing
    // and then it converts to hex string
    const hashBuffer = await crypto.subtle.digest('SHA-256', tokenBytes);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2,'0')).join('');
    return hashHex;
}

// Utilities for URLs

// help -- changing base64 to base64url (URL-safe)
function base64ToBase64Url(base64) {
    return base64
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=/g, '');
}


// shareable room url for users
function buildRoomURL(roomId, encryptionKey) {
    const keyString = keyToString(encryptionKey);
    const urlSafeKey = base64ToBase64Url(keyString);
    const baseURL = window.location.origin;

    return `${baseURL}/room.html?room=${roomId}#key=${urlSafeKey}`;
}


// admin room URL (with admin token)
function buildAdminURL(roomId, encryptionKey, adminToken) {
    const keyString = keyToString(encryptionKey);
    const urlSafeKey = base64ToBase64Url(keyString);
    const urlSafeAdmin = base64ToBase64Url(adminToken);
    const baseURL = window.location.origin;

    return `${baseURL}/room.html?room=${roomId}#key=${urlSafeKey}&admin=${urlSafeAdmin}`;
}


// changing base64url back to base64
function base64UrlToBase64(base64url) {
    let base64 = base64url
        .replace(/-/g, '+')
        .replace(/_/g, '/');

    // Add back padding if needed
    while (base64.length % 4 !== 0) {
        base64 += '=';
    }

    return base64;
}

// now, parsing to get the key and admin token
function parseRoomURL() {
    const urlParams = new URLSearchParams(window.location.search);
    const roomId = urlParams.get('room');

    const fragment = window.location.hash.substring(1);
    const fragmentParams = new URLSearchParams(fragment);

    const urlSafeKey = fragmentParams.get('key');
    const urlSafeAdmin = fragmentParams.get('admin');

    // conversion  from base64url to base64
    const keyString = urlSafeKey ? base64UrlToBase64(urlSafeKey) : null;
    const adminToken = urlSafeAdmin ? base64UrlToBase64(urlSafeAdmin) : null;

    const encryptionKey = keyString ? stringToKey(keyString) : null;

    return {
        roomId,
        encryptionKey,
        adminToken
    };
}
