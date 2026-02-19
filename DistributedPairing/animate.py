import json
import sys
import networkx as nx
import matplotlib.pyplot as plt
from matplotlib.animation import FuncAnimation

# Color mapping based on node state
STATE_COLORS = {
    "SINGLE": "#B0BEC5",   # Grey
    "PROPOSER": "#FFB300", # Yellow
    "LISTENER": "#42A5F5", # Blue
    "MATCHED": "#66BB6A"   # Green
}

# 100 milliseconds represented in nanoseconds for event binning
TIME_WINDOW_NS = 100_000_000 

class ConcurrentSimulationVisualizer:
    def __init__(self, json_file):
        with open(json_file, 'r') as f:
            raw_events = json.load(f)

        self.graph = nx.Graph()
        self.node_states = {}
        self.matched_edges = set()
        
        # Extract topology from the first event
        init_event = raw_events.pop(0)
        if init_event.get("type") != "INIT":
            raise ValueError("First event must be INIT")
            
        self.graph.add_nodes_from(init_event.get("nodes", []))
        self.graph.add_edges_from(init_event.get("edges", []))
        
        # Calculate fixed positions for nodes to maintain static layout
        self.pos = nx.spring_layout(self.graph, seed=42)
        
        for node in self.graph.nodes:
            self.node_states[node] = "SINGLE"

        # Group the remaining raw events into concurrent frames
        self.frames = self.group_events_by_time(raw_events)

        self.fig, self.ax = plt.subplots(figsize=(10, 8))
        self.fig.canvas.manager.set_window_title('Concurrent Maximal Matching Execution')

    def group_events_by_time(self, events):
        """
        Batches strictly sequential events into concurrent frames 
        based on a nanosecond time delta to visualize parallelism.
        """
        if not events:
            return []

        # Ensure chronological order
        events.sort(key=lambda e: e.get("timestamp", 0))
        
        batched_frames = []
        current_batch = []
        window_start = events[0].get("timestamp", 0)

        for event in events:
            ts = event.get("timestamp", 0)
            if ts - window_start <= TIME_WINDOW_NS:
                current_batch.append(event)
            else:
                batched_frames.append(current_batch)
                current_batch = [event]
                window_start = ts
                
        if current_batch:
            batched_frames.append(current_batch)
            
        return batched_frames

    def update_frame(self, frame_idx):
        self.ax.clear()
        
        if frame_idx >= len(self.frames):
            return

        # Fetch the entire batch of events occurring concurrently
        current_events = self.frames[frame_idx]
        active_messages = []
        
        state_changes_count = 0
        message_count = 0
        match_count = 0

        # Apply all logic for this time window
        for event in current_events:
            event_type = event.get("type")

            if event_type == "STATE_CHANGE":
                node = event.get("node")
                state = event.get("state")
                if node is not None and state is not None:
                    self.node_states[node] = state
                    state_changes_count += 1
                
            elif event_type == "MSG_SENT":
                msg = event.get("msg", {})
                sender = msg.get("sender")
                target = msg.get("target")
                mtype = msg.get("type")
                if sender is not None and target is not None:
                    active_messages.append((sender, target, mtype))
                    message_count += 1
                
            elif event_type == "MATCHED":
                node = event.get("node")
                partner = event.get("partner")
                if node is not None and partner is not None:
                    self.node_states[node] = "MATCHED"
                    if node != partner:
                        self.matched_edges.add(tuple(sorted((node, partner))))
                    match_count += 1

        # Build dynamic title summarizing concurrent actions
        title_text = f"Time Window {frame_idx + 1}/{len(self.frames)} | "
        title_text += f"{message_count} Msgs | {state_changes_count} State Updates | {match_count} Matches"
        self.ax.set_title(title_text, fontsize=14)
        
        # Base graph rendering
        nx.draw_networkx_edges(self.graph, self.pos, ax=self.ax, edge_color='#E0E0E0', width=1.0)
        nx.draw_networkx_edges(self.graph, self.pos, ax=self.ax, edgelist=list(self.matched_edges), edge_color='#66BB6A', width=3.0)

        colors = [STATE_COLORS[self.node_states[n]] for n in self.graph.nodes]
        nx.draw_networkx_nodes(self.graph, self.pos, ax=self.ax, node_color=colors, node_size=600, edgecolors='black')
        nx.draw_networkx_labels(self.graph, self.pos, ax=self.ax, font_color='black', font_weight='bold')

        # Render all concurrent messages in this frame simultaneously
        for sender, target, mtype in active_messages:
            x_sender, y_sender = self.pos[sender]
            x_target, y_target = self.pos[target]
            
            self.ax.annotate(
                mtype,
                xy=(x_target, y_target), xycoords='data',
                xytext=(x_sender, y_sender), textcoords='data',
                arrowprops=dict(arrowstyle="->", color="red", shrinkA=15, shrinkB=15, lw=2),
                bbox=dict(boxstyle="round,pad=0.2", fc="white", ec="red", alpha=0.8),
                ha='center', va='center', fontsize=9, color='red', weight='bold'
            )

        self.ax.axis('off')

    def animate(self):
        # We increase the interval so you have time to digest the concurrent batch 
        # happening on screen before it moves to the next window.
        anim = FuncAnimation(
            self.fig, self.update_frame, 
            frames=len(self.frames), 
            interval=300, 
            repeat=False
        )
        plt.show()

if __name__ == "__main__":
    if len(sys.argv) > 1:
        json_path = sys.argv[1]
    else:
        json_path = 'simulation_events.json'
        
    try:
        vis = ConcurrentSimulationVisualizer(json_path)
        vis.animate()
    except FileNotFoundError:
        print(f"Error: Could not find '{json_path}'. Run the Go simulation first.")