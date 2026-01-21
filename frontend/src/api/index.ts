import api from './client';

export interface User {
  id: number;
  username: string;
  email: string;
  full_name: string;
  role: 'admin' | 'user';
  is_active: boolean;
  created_at: string;
  last_login?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  user: User;
}

export const authAPI = {
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response = await api.post<LoginResponse>('/login', credentials);
    return response.data;
  },

  logout: async (): Promise<void> => {
    await api.post('/logout');
  },

  checkAuth: async (): Promise<{ authenticated: boolean; user: User }> => {
    const response = await api.get<{ authenticated: boolean; user: User }>('/auth/check');
    return response.data;
  },
};

export interface Share {
  id: number;
  share_name: string;
  display_name: string;
  path: string;
  comment: string;
  browseable: boolean;
  readonly: boolean;
  guest_ok: boolean;
  is_active: boolean;
}

export const sharesAPI = {
  list: async (): Promise<Share[]> => {
    const response = await api.get<{ success: boolean; shares: Share[] }>('/shares');
    return response.data.shares;
  },

  get: async (id: number): Promise<Share> => {
    const response = await api.get<{ success: boolean; share: Share }>(`/shares/${id}`);
    return response.data.share;
  },

  create: async (share: Partial<Share>): Promise<{ success: boolean; share_id: number }> => {
    const response = await api.post<{ success: boolean; share_id: number }>('/shares', share);
    return response.data;
  },

  update: async (id: number, share: Partial<Share>): Promise<{ success: boolean }> => {
    const response = await api.put<{ success: boolean }>(`/shares/${id}`, share);
    return response.data;
  },

  delete: async (id: number): Promise<{ success: boolean }> => {
    const response = await api.delete<{ success: boolean }>(`/shares/${id}`);
    return response.data;
  },

  toggle: async (id: number): Promise<{ success: boolean; is_active: boolean }> => {
    const response = await api.post<{ success: boolean; is_active: boolean }>(`/shares/${id}/toggle`);
    return response.data;
  },
};

export interface VirtualMachine {
  id: number;
  name: string;
  description: string;
  uuid: string;
  cpu_cores: number;
  ram_mb: number;
  disk_path: string;
  disk_size_gb: number;
  disk_format: string;
  status: 'stopped' | 'running' | 'paused' | 'error';
  spice_port: number;
}

export const vmsAPI = {
  list: async (): Promise<VirtualMachine[]> => {
    const response = await api.get<{ success: boolean; vms: VirtualMachine[] }>('/vms');
    return response.data.vms;
  },

  get: async (id: number): Promise<VirtualMachine> => {
    const response = await api.get<{ success: boolean; vm: VirtualMachine }>(`/vms/${id}`);
    return response.data.vm;
  },

  create: async (vm: Partial<VirtualMachine>): Promise<{ success: boolean; vm_id: number }> => {
    const response = await api.post<{ success: boolean; vm_id: number }>('/vms', vm);
    return response.data;
  },

  update: async (id: number, vm: Partial<VirtualMachine>): Promise<{ success: boolean }> => {
    const response = await api.put<{ success: boolean }>(`/vms/${id}`, vm);
    return response.data;
  },

  delete: async (id: number): Promise<{ success: boolean }> => {
    const response = await api.delete<{ success: boolean }>(`/vms/${id}`);
    return response.data;
  },

  start: async (id: number): Promise<{ success: boolean }> => {
    const response = await api.post<{ success: boolean }>(`/vms/${id}/start`);
    return response.data;
  },

  stop: async (id: number, force?: boolean): Promise<{ success: boolean }> => {
    const response = await api.post<{ success: boolean }>(`/vms/${id}/stop`, { force });
    return response.data;
  },

  restart: async (id: number): Promise<{ success: boolean }> => {
    const response = await api.post<{ success: boolean }>(`/vms/${id}/restart`);
    return response.data;
  },

  getStatus: async (id: number): Promise<{ success: boolean; status: string }> => {
    const response = await api.get<{ success: boolean; status: string }>(`/vms/${id}/status`);
    return response.data;
  },
};

export interface SystemStats {
  cpu: {
    model: string;
    cores: number;
    architecture: string;
    usage: number;
    load_avg: {
      '1min': number;
      '5min': number;
      '15min': number;
    };
  };
  memory: {
    total: number;
    used: number;
    available: number;
    usage_percent: number;
    total_formatted: string;
    used_formatted: string;
    available_formatted: string;
  };
  swap: {
    total: number;
    used: number;
    usage_percent: number;
    total_formatted: string;
    used_formatted: string;
  };
  uptime: {
    seconds: number;
    formatted: string;
  };
  network: Array<{
    name: string;
    status: string;
    is_up: boolean;
    ip: string;
    mac: string;
    rx_formatted: string;
    tx_formatted: string;
  }>;
}

export const systemAPI = {
  getStats: async (): Promise<SystemStats> => {
    const response = await api.get<SystemStats>('/system/stats');
    return response.data;
  },
};

export const usersAPI = {
  list: async (): Promise<User[]> => {
    const response = await api.get<{ success: boolean; users: User[] }>('/users');
    return response.data.users;
  },

  create: async (user: Partial<User> & { password: string }): Promise<{ success: boolean; user_id: number }> => {
    const response = await api.post<{ success: boolean; user_id: number }>('/users', user);
    return response.data;
  },

  update: async (id: number, user: Partial<User>): Promise<{ success: boolean }> => {
    const response = await api.put<{ success: boolean }>(`/users/${id}`, user);
    return response.data;
  },

  delete: async (id: number): Promise<{ success: boolean }> => {
    const response = await api.delete<{ success: boolean }>(`/users/${id}`);
    return response.data;
  },
};

export interface TerminalResponse {
  success: boolean;
  output: string;
  error: boolean;
  cwd: string;
}

export const terminalAPI = {
  execute: async (command: string, cwd?: string): Promise<TerminalResponse> => {
    const response = await api.post<TerminalResponse>('/terminal/execute', { command, cwd });
    return response.data;
  },
};

export interface NetworkInterface {
  name: string;
  display_name: string;
  type: string;
  status: string;
  is_up: boolean;
  ip: string;
  ipv6: string;
  subnet: string;
  gateway: string;
  mac: string;
  mtu: number;
  speed: string;
  duplex: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_formatted: string;
  tx_formatted: string;
  rx_session: number;
  tx_session: number;
  rx_session_formatted: string;
  tx_session_formatted: string;
  rx_speed: number;
  tx_speed: number;
  rx_speed_formatted: string;
  tx_speed_formatted: string;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
  rx_dropped: number;
  tx_dropped: number;
  driver: string;
  is_virtual: boolean;
  is_wireless: boolean;
}

export interface NetworkProcess {
  pid: number;
  name: string;
  user: string;
  connections: number;
  rx_bytes: number;
  tx_bytes: number;
  rx_speed: number;
  tx_speed: number;
  rx_speed_formatted: string;
  tx_speed_formatted: string;
}

export interface NetworkConnection {
  protocol: string;
  state: string;
  recv_q: string;
  send_q: string;
  local: string;
  remote?: string;
  process?: string;
}

export interface BandwidthHistoryEntry {
  timestamp: string;
  total_rx_speed: number;
  total_tx_speed: number;
  interfaces_rx: Record<string, number>;
  interfaces_tx: Record<string, number>;
}

export interface NetworkEvent {
  id: number;
  timestamp: string;
  type: string;
  interface: string;
  description: string;
  details: string;
}

export const networkAPI = {
  getInterfaces: async (): Promise<{ interfaces: NetworkInterface[]; session_start: string }> => {
    const response = await api.get<{ success: boolean; interfaces: NetworkInterface[]; session_start: string }>('/network/interfaces');
    return response.data;
  },

  getInterface: async (name: string): Promise<NetworkInterface> => {
    const response = await api.get<{ success: boolean; interface: NetworkInterface }>(`/network/interfaces/${name}`);
    return response.data.interface;
  },

  toggleInterface: async (name: string, action: 'up' | 'down', force?: boolean): Promise<{
    success: boolean;
    message?: string;
    error?: string;
    is_active_conn?: boolean;
    warning?: string;
  }> => {
    const response = await api.post(`/network/interfaces/${name}/toggle`, { action, force });
    return response.data;
  },

  getBandwidth: async (): Promise<{
    interfaces: NetworkInterface[];
    total_rx: number;
    total_tx: number;
    total_rx_formatted: string;
    total_tx_formatted: string;
    total_rx_speed: number;
    total_tx_speed: number;
    total_rx_speed_fmt: string;
    total_tx_speed_fmt: string;
    history: BandwidthHistoryEntry[];
  }> => {
    const response = await api.get('/network/bandwidth');
    return response.data;
  },

  getHistory: async (): Promise<BandwidthHistoryEntry[]> => {
    const response = await api.get<{ success: boolean; history: BandwidthHistoryEntry[] }>('/network/history');
    return response.data.history;
  },

  getEvents: async (): Promise<NetworkEvent[]> => {
    const response = await api.get<{ success: boolean; events: NetworkEvent[] }>('/network/events');
    return response.data.events;
  },

  getProcesses: async (): Promise<NetworkProcess[]> => {
    const response = await api.get<{ success: boolean; processes: NetworkProcess[] }>('/network/processes');
    return response.data.processes;
  },

  getConnections: async (): Promise<NetworkConnection[]> => {
    const response = await api.get<{ success: boolean; connections: NetworkConnection[] }>('/network/connections');
    return response.data.connections;
  },

  resetSession: async (): Promise<{ success: boolean }> => {
    const response = await api.post<{ success: boolean }>('/network/session/reset');
    return response.data;
  },
};

