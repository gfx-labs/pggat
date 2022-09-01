# frozen_string_literal: true
require_relative 'spec_helper'

describe "Miscellaneous" do
  let(:processes) { Helpers::Pgcat.single_shard_setup("sharded_db", 5) }
  after do
    processes.all_databases.map(&:reset)
    processes.pgcat.shutdown
  end

  describe "Extended Protocol handling" do
    it "does not send packets that client does not expect during extended protocol sequence" do
      new_configs = processes.pgcat.current_config

      new_configs["general"]["connect_timeout"] = 500
      new_configs["general"]["ban_time"] = 1
      new_configs["general"]["shutdown_timeout"] = 1
      new_configs["pools"]["sharded_db"]["users"]["0"]["pool_size"] = 1

      processes.pgcat.update_config(new_configs)
      processes.pgcat.reload_config

      25.times do
        Thread.new do
          conn = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
          conn.async_exec("SELECT pg_sleep(5)") rescue PG::SystemError
        ensure
          conn&.close
        end
      end

      sleep(0.5)
      conn_under_test = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
      stdout, stderr = with_captured_stdout_stderr do
        15.times do |i|
          conn_under_test.async_exec("SELECT 1") rescue PG::SystemError
          conn_under_test.exec_params("SELECT #{i} + $1", [i]) rescue PG::SystemError
          sleep 1
        end
      end

      raise StandardError, "Libpq got unexpected messages while idle" if stderr.include?("arrived from server while idle")
    end
  end

  describe "Pool recycling after config reload" do
    let(:processes) { Helpers::Pgcat.three_shard_setup("sharded_db", 5) }

    it "should update pools for new clients and clients that are no longer in transaction" do
      server_conn = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
      server_conn.async_exec("BEGIN")

      # No config change yet, client should set old configs
      current_datebase_from_pg = server_conn.async_exec("SELECT current_database();")[0]["current_database"]
      expect(current_datebase_from_pg).to eq('shard0')

      # Swap shards
      new_config = processes.pgcat.current_config
      shard0 = new_config["pools"]["sharded_db"]["shards"]["0"]
      shard1 = new_config["pools"]["sharded_db"]["shards"]["1"]
      new_config["pools"]["sharded_db"]["shards"]["0"] = shard1
      new_config["pools"]["sharded_db"]["shards"]["1"] = shard0

      # Reload config
      processes.pgcat.update_config(new_config)
      processes.pgcat.reload_config
      sleep 0.5

      # Config changed but transaction is in progress, client should set old configs
      current_datebase_from_pg = server_conn.async_exec("SELECT current_database();")[0]["current_database"]
      expect(current_datebase_from_pg).to eq('shard0')
      server_conn.async_exec("COMMIT")

      # Transaction finished, client should get new configs
      current_datebase_from_pg = server_conn.async_exec("SELECT current_database();")[0]["current_database"]
      expect(current_datebase_from_pg).to eq('shard1')

      # New connection should get new configs
      server_conn.close()
      server_conn = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
      current_datebase_from_pg = server_conn.async_exec("SELECT current_database();")[0]["current_database"]
      expect(current_datebase_from_pg).to eq('shard1')
    end
  end

  describe "Clients closing connection in the middle of transaction" do
    it "sends a rollback to the server" do
      conn = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
      conn.async_exec("SET SERVER ROLE to 'primary'")
      conn.async_exec("BEGIN")
      conn.close

      expect(processes.primary.count_query("ROLLBACK")).to eq(1)
      expect(processes.primary.count_query("DISCARD ALL")).to eq(1)
    end
  end

  describe "Server version reporting" do
    it "reports correct version for normal and admin databases" do
      server_conn = PG::connect(processes.pgcat.connection_string("sharded_db", "sharding_user"))
      expect(server_conn.server_version).not_to eq(0)
      server_conn.close

      admin_conn = PG::connect(processes.pgcat.admin_connection_string)
      expect(admin_conn.server_version).not_to eq(0)
      admin_conn.close
    end
  end
end
