import asyncio
import json
import websockets


channels = {}

async def relay(websocket, path):
    # Register client
    channel_name = None
    channel = None
    is_sender = False
    is_receiver = False
    try:
        # Wait for channel selection message
        message = await websocket.recv()
        data = json.loads(message)
        if data["command"] == "join":
            channel_name = data["channel"]
            if channel_name not in channels:
                channels[channel_name] = {"senders": set(), "receivers": set()}
            channel = channels[channel_name]
            print(f"Client joined channel {channel_name}")
        else:
            await websocket.close(reason="Must join a channel first.")
            return

# Wait for role selection message
        while not is_sender and not is_receiver:
            message = await websocket.recv()
            data = json.loads(message)
            if data["command"] == "select-role":
                if data["role"] == "sender":
                    if len(channel["senders"]) == 0:
                        channel["senders"].add(websocket)
                        is_sender = True
                        print("Client joined as sender")
                    else:
                        await websocket.send(json.dumps({"type": "ERROR", "message": "Cannot join as sender: sender already connected"}))
                elif data["role"] == "receiver":
                    if len(channel["receivers"]) == 0:
                        channel["receivers"].add(websocket)
                        is_receiver = True
                        print("Client joined as receiver")
                    else:
                        await websocket.send(json.dumps({"type": "ERROR", "message": "Cannot join as receiver: receiver already connected"}))
            else:
                await websocket.close(reason="Invalid role selection.")
                return

        # Receive message from sender or send message to receiver
        while True:
            if is_receiver:
                message = await websocket.recv()
                print("Received message from sender:", message)

                # Forward message to receiver
                if len(channel["senders"]) > 0:
                    sender = next(iter(channel["senders"])) if len(channel["senders"]) > 0 else None
                    if sender is not None:
                        await sender.send(message)
                    else:
                        print("No senders connected to forward message.")
                        break
                else:
                    print("No senders connected to forward message.")
                    break
            elif is_sender:
                message = await websocket.recv()

                # Forward message to receiver
                if len(channel["receivers"]) > 0:
                    receiver = next(iter(channel["receivers"])) if len(channel["receivers"]) > 0 else None
                    if receiver is not None:
                        await receiver.send(message)
                    else:
                        print("No receivers connected to forward message.")
                        break
                else:
                    print("No receivers connected to forward message.")
                    break

    finally:
        # Unregister client
        if is_sender:
            channel["senders"].remove(websocket)
        elif is_receiver:
            channel["receivers"].remove(websocket)

        # Remove channel if empty
        if channel_name is not None and len(channel["senders"]) == 0 and len(channel["receivers"]) == 0:
            channels.pop(channel_name)
            print(f"Channel {channel_name} removed.")

async def start_relay():
    async with websockets.serve(relay, '99.33.36.109', 80, max_size=1024*1024*1024*1024):
        await asyncio.Future()  # Run forever

asyncio.get_event_loop().run_until_complete(start_relay())
