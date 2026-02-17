//random key generation - gives out 32 random bytes = 256 (supposed to be secure) as key. Why: https://www.ssldragon.com/blog/256-bit-encryption/, https://www.clickssl.net/blog/256-bit-encryption, https://www.newsoftwares.net/blog/why-256-bit-isnt-always-stronger-than-128-bit/, https://en.wikipedia.org/wiki/Advanced_Encryption_Standard 
function generateEncryptionKey(){
    return nacl.randomBytes(32);
}

//The keygen will be used for everything that has to be encrypted (urls, messages, timestamps, etc.)

//https://en.wikipedia.org/wiki/Base64
function KeyToString(key){
    return nacl.util.encodeBase64(key);
}

//https://stackoverflow.com/questions/38718202/how-to-use-uint8array-uint16array-uin32array, https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Uint8Array
function stringToKey(keyString){
    try{
        //the decodeBase64 makes base64 strings to bytes
        return nacl.util.decodeBase64(keyString);
    } catch(error){
        console.error('Invalid encryption key format');
        return null;
    }
}

//messages
//treat messages as objects and turn them into json strings.Then turn the json strings to bytes via Uint8Array
function encryptMessage(messageObj,key){
    const messageJSON = JSON.stringify(messageObj);
    const messageBytes = nacl.util.decodeUTF8(messageJSON);
}