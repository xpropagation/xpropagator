#
# MIT License
#
# Copyright (c) 2026 Roman Bielyi
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

grpcurl -plaintext -d '{
    "req_id": "1",
    "ephem_type": "EphemJ2K",
    "common_time_grid": {
      "time_start_utc": "2025-12-18T00:00:00Z",
      "time_end_utc": "2025-12-28T00:00:00Z",
      "dynamic_time_step": true
    },
    "tasks": [
    {
      "task_id": 10,
      "sat": {
        "norad_id": 65271,
        "name": "X-37B Orbital Test Vehicle 8 (OTV 8)",
        "tle_ln1": "1 65271U 25183A   25282.36302114 0.00010000  00000-0  55866-4 0    07",
        "tle_ln2": "2 65271  48.7951   8.5514 0002000  85.4867 277.3551 15.78566782    05"
      }
    },
    {
      "task_id": 20,
      "sat": {
        "sat_id": 2000,
        "name": "Satellite B",
        "tle_ln1": "1 49220U 21089B   24290.21456789  .00014567  00000-0  62458-3 0  9991",
        "tle_ln2": "2 49220  53.0021 320.8765 0078456  42.6543 317.8845 14.87654321987654"
      }
    },
    {
      "task_id": 30,
      "sat": {
        "sat_id": 3000,
        "name": "Satellite C",
        "tle_ln1": "1 60123U 24150C   24290.84567890  .00000023  00000-0  15987-5 0  9993",
        "tle_ln2": "2 60123  28.5123 210.2345 0019876 102.3456 257.6543 12.34567890123456"
      }
    }
  ]
}' localhost:50051 api.v1.Propagator.Ephem