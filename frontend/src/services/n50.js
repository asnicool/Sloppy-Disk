import axios from 'axios'

const API_BASE = '/api'

export const n50Service = {
  // Get N50 status
  async getStatus() {
    try {
      const response = await axios.get(`${API_BASE}/n50/status`)
      return response.data
    } catch (error) {
      console.error('Failed to get N50 status:', error)
      throw error
    }
  },

  // Power on N50
  async powerOn() {
    try {
      const response = await axios.post(`${API_BASE}/n50/power/on`)
      return response.data
    } catch (error) {
      console.error('Failed to power on N50:', error)
      throw error
    }
  },

  // Power off (standby) N50
  async powerOff() {
    try {
      const response = await axios.post(`${API_BASE}/n50/power/off`)
      return response.data
    } catch (error) {
      console.error('Failed to power off N50:', error)
      throw error
    }
  },

  // Set input source
  async setInput(input) {
    try {
      const response = await axios.post(`${API_BASE}/n50/input/${input}`)
      return response.data
    } catch (error) {
      console.error('Failed to set N50 input:', error)
      throw error
    }
  },

  // Get available inputs
  async getAvailableInputs() {
    try {
      const response = await axios.get(`${API_BASE}/n50/inputs`)
      return response.data
    } catch (error) {
      console.error('Failed to get N50 inputs:', error)
      throw error
    }
  }
}
