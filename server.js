//Settings and dependencies

const express = require('express');
const { WebSocketServer } = require('ws');
const Database = require('better-sqlite3');
const multer = require('multer');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const http = require('http');

//Settings from env file or env vars on Coolify. 
const PORT = process.env.PORT || 3000;
const STORAGE_PATH = process.env.STORAGE_PATH || './uploads';
const MAX_FILE_SIZE = parseInt(process.env.MAX_FILE_SIZE) || 10485760; // 10MB
const MAX_IMAGES_PER_ROOM = parseInt(process.env.MAX_IMAGES_PER_ROOM) || 20;
const ROOM_EXPIRY_DAYS = parseInt(process.env.ROOM_EXPIRY_DAYS) || 30;

//Database setup
const db = new Database('database.db');

db.exec(`
    CREATE TABLE IF NOT EXISTS rooms (
    room_id TEXT PRIMARY KEY,
    admin_token_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
    image_count INTEGER DEFAULT 0);
    
    CREATE TABLE IF NOT EXISTS files (
    file_id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    filepath TEXT NOT NULL,
    filename TEXT NOT NULL,
    size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(room_id) ON DELETE CASCADE);
    
    CREATE INDEX IF NOT EXISTS idx_room_files ON files(room_id);`);

    console.log('Done with database initialization');

    // Helpers

    // random names for everyone
    function generateId(length = 16) {
        return crypto.randomBytes(length).toString('hex');
    }
    //encrypt admin token for storage
    function hashToken(token) {
        return crypto.createHash('sha256').update(tokenBytes).digest('hex');
    }

    //Does the file for uploads exist?
    if (!fs.existsSync(STORAGE_PATH)) {
        fs.mkdirSync(STORAGE_PATH, {recursive: true});
        console.log ('Folder for uploads created');
    }

    //express (service?)
    const app = express();
    const server = http.createServer(app);

    
    app.use(express.json({ limit: '15mb'}));
    app.use(express.static('public')); //So that statics can be served

    //endpoints

    //new room
    app.post('/api/create-room', (req,res) => {
        const {room_id, admin_token_hash} = req.body;

        if (!room_id || !admin_token_hash) {
            return res.status(400).json({error: 'No room_id or admin_token_hash'});
        }

        try {
            const stmt = db.prepare(`INSERT INTO rooms (room_id, admin_token_hash) VALUES (?, ?)`);
            stmt.run(room_id, admin_token_hash);

            console.log(`Room created: ${room_id}`);
            res.json({success: true, room_id});
        } catch (error) {
            console.error('Error creating room:', error);
            res.status(500).json({error: 'Failed to create room'});
        }
    });

    //encrypted file upload
    app.post('/api/upload', (req,res) => {
        const {room_id, encrypted_data, metadata} = req.body;
        
        if (!room_id || !encrypted_data || !metadata)
            return res.status(400).json({error: 'Missing required fields'});
        
        try {
            const room = db.prepare('SELECT * FROM rooms WHERE room_id = ?').get(room_id);
            if (!room) {
                return res.status(404).json({error: 'Room not found'});
            }
            if (room.image_count >= MAX_IMAGES_PER_ROOM) {
                return res.status(403).json({error: 'Room image limit reached'});
            }
            if (metadata.size > MAX_FILE_SIZE) {
                return res.status(413).json({error:'File too large'});
            }

            //encrypt image
            const file_id = generateId();
            const roomDir = path.join(STORAGE_PATH, room_id);
            if (!fs.existsSync(roomDir)) {
                fs.mkdirSync(roomDir, {recursive: true});
            }

            //save on room & database
            const filepath = path.join(roomDir, `${file_id}.enc`);
            const buffer = Buffer.from(encrypted_data, 'base64');
            fs.writeFileSync(filepath, buffer);

            const stmt = db.prepare(`
                INSERT INTO files (file_id, room_id, filepath, filename, size, mime_type)
                VALUES (?, ?, ?, ?, ?, ?)`);
            stmt.run(file_id, room_id, filepath, metadata.filename, metadata.size, metadata.mime_type);

            db.prepare('UPDATE rooms SET image_count = image_count + 1, last_activity = CURRENT_TIMESTAMP WHERE room_id = ?')
                .run(room_id);
            
            console.log(`Image uploaded: ${file_id} to room ${room_id}`);
            res.json({ success: true, file_id, metadata});
        } catch (error) {
            console.error('Error uploading file:', error);
            res.status(500).json({error: 'Failed to upload file'});
        }
    });
    

    //download-availability for images
    app.get('/api/download/:file_id', (req, res) => {
        const {file_id} = req.params;

        try{
            const file = db.prepare('SELECT * FROM files WHERE file_id = ?').get(file_id);
            if(!file) {
                return res.status(404).json({error:'File not found'});
            }
            if(!fs.existsSync(file.filepath)) {
                return res.status(404).json({error:'File data not found'});
            }

            const fileData = fs.readFileSync(file.filepath);
            res.json({
                success: true,
                encrypted_data: fileData.toString('base64'),
                metadata:{
                    filename: file.filename,
                    size:file.size,
                    mime_type: file.mime_type
                }
            });
        }   catch (error) {
            console.error('Error downloading file:', error);
            res.status(500).json({ error: 'Failed to download file'});
        }
    });

    //room deleting
    app.delete('/api/room/:room_id',(req,res) => {
        const {room_id} = req.params;
        const {admin_token} = req.body;

        if (!admin_token) {
            return res.status(400).json({error: 'Missing admin_token'});
        }

        try {
            const room = db.prepare('SELECT * FROM rooms WHERE room_id = ?').get(room_id);
            if(!room) {
                return res.status(404).json({error: 'Room not found'});
            }
            //admin verification
            if(hashToken(admin_token) !== room.admin_token_hash) {
                return res.status(403).json({error:'Invalid admin token'});
            }

            //delete files from server disk
            const roomDir = path.join(STORAGE_PATH, room_id);
            if (fs.existsSync(roomDir)) {
                fs.rmSync(roomDir, {recursive: true ,force: true});
            }

            //delete entry from db
            db.prepare('DELETE FROM rooms WHERE room_id = ?').run(room_id);

            console.log(`Room ${room_id} was deleted`);

            //notification for connected clients
            broadcastToRoom(room_id, {type: 'room_deleted'});

            res.json({success: true});
        }   catch(error) {
            console.error('Error deleting room:', error);
            res.status(500).json({error: 'Failed to delete room'});
        }
    });

    //Websocket

    const wss = new WebSocketServer({server});

    //storing active connections
    const rooms = new Map();
    wss.on('connection',(ws) => {
        console.log('New WebSocket connection');

        ws.on('message', (message) => {
            try {
                const data = JSON.parse(message);

                switch (data.type) {
                    case 'join':
                        handleJoin(ws,data);
                        break;
                    case 'message':
                        handleMessage(ws, data);
                        break;
                    case 'typing_start':
                        broadcastToRoom(data.room_id,{
                            type:'typing_start',
                            username: data.username
                        }, ws);
                        break;
                    case 'typing_stop':
                        broadcastToRoom(data.room_id, {
                            type: 'typing_stop',
                            username: data.username
                        }, ws);
                        break;
                    case 'image_notification':
                        handleImageNotification(ws, data);
                        break;
                    default:
                        console.log('Unknown message type:', data.type);
                    }
                }   catch(error) {
                console.error('Error handling WebSocket message:', error);
                }
            });
            ws.on('close', () => {
                handleDisconnect(ws);
            });
        });
    
        function handleJoin(ws, data) {
            const{room_id} = data;

            //Does room exist?
            const room = db.prepare('SELECT * FROM rooms WHERE room_id = ?').get(room_id);
            if(!room){
                ws.send(JSON.stringify({type: 'error', message: 'Room not found'}));
                return;
            }

            //Add to room
            ws.room_id = room_id;
            if(!rooms.has(room_id)){
                rooms.set(room_id, []);
            }
            rooms.get(room_id).push(ws);

            //latest activity
            db.prepare('UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE room_id = ?').run(room_id);

            //Confirm
            const count = rooms.get(room_id).length;
            ws.send(JSON.stringify({type: 'joined', count}));

            //notify participants
            broadcastToRoom(room_id, {type: 'user_joined', count}, ws);

            console.log(`User joined room ${room_id} (${count} users)`);
        }

        function handleMessage(ws, data) {
            const {room_id, encrypted_data} = data;

            //update latest activity
            db.prepare('UPDATE rooms SET last_activity = CURRENT_TIMESTAMP WHERE room_id = ?').run(room_id);

            
            broadcastToRoom(room_id, {type: 'message', encrypted_data});
        }

        function handleImageNotification(ws, data) {
            const {room_id, file_id, metadata} = data;

            
            broadcastToRoom(room_id, {type: 'new_image', file_id, metadata});
        }

        function handleDisconnect(ws) {
            if (ws.room_id && rooms.has(ws.room_id)) {
                const room = rooms.get(ws.room_id);
                const index = room. indexOf(ws);
                if (index > -1) {
                    room.splice(index, 1);
                }

                const count = room.length;

                if (count === 0) {
                    rooms.delete(ws.room_id);
                } else {
                    broadcastToRoom(ws.room_id, {type: 'user_left',count});
                }

                console.log(`User left ${ws.room_id}(${count} users remaining)`);
            }
        }

        function broadcastToRoom (room_id, message, exclude = null) {
            if (rooms.has(room_id)) {
                const payload = JSON.stringify(message);
                rooms.get(room_id).forEach((client) => {
                    if (client !== exclude && client.readyState === 1) { // 1/0 = Open/Close
                        client.send(payload);
                    } 
                });
            }
        }

//Daily cleaning

function cleanupOldRooms() {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - ROOM_EXPIRY_DAYS);
    const cutoff = cutoffDate.toISOString();

    const oldRooms = db.prepare('SELECT room_id FROM rooms WHERE last_activity < ?').all(cutoff);

    oldRooms.forEach((room) => {
        //remove any files from server disk
        const roomDir = path.join(STORAGE_PATH, room.room_id);
        if (fs.existsSync(roomDir)) {
            fs.rmSync(roomDir, {recursive: true, force: true});
        }

        //delete entries from db
        db.prepare('DELETE FROM rooms WHERE room_id = ?').run(room.room_id);

        console.log(`Cleaned up old room: ${room.room_id}`);
    });

    if (oldRooms.length > 0) {
        console.log(`Cleaned up ${oldRooms.length} old rooms`);
    }
}

//do it daily
setInterval (cleanupOldRooms, 24 * 60 *60 * 1000);

//server start
server.listen(PORT, () => {
    console.log(`
     ╔════════════════════════════════════════╗
     ║                                        ║
     ║       🪑 BenchTalks Server v1.0        ║
     ║                                        ║
     ║  Server running on port ${PORT}           ║
     ║  http://localhost:${PORT}                 ║
     ║                                        ║
     ╚════════════════════════════════════════╝   
    `);
});