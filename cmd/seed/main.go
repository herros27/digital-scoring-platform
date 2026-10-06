package main

import (
	"fmt"
	"log"
	"os"

	"github.com/herros27/digital-scoring/internal/config"
	"github.com/herros27/digital-scoring/internal/store"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🥋 DIGITAL SCORING PENCAK SILAT — DATABASE SEEDER")
	fmt.Println("================================================================================")

	cfg := config.Load()

	db := store.NewDB(cfg.DBURL)
	defer db.Close()

	fmt.Println("🔄 Memastikan skema database up-to-date...")
	if err := store.RunMigrations(db); err != nil {
		log.Fatalf("❌ Migrasi database gagal: %v", err)
	}

	fmt.Println("🌱 Mengisi data awal (seeding data)...")
	summary, err := store.SeedData(db)
	if err != nil {
		log.Fatalf("❌ Seeding database gagal: %v", err)
	}

	fmt.Println("\n✅ SEEDING DATABASE SELESAI!")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("🏆 Kompetisi ID : %s\n", summary.CompetitionID)
	fmt.Printf("🏟️ Gelanggang A  : %s\n", summary.ArenaAID)
	fmt.Printf("🏟️ Gelanggang B  : %s\n", summary.ArenaBID)
	fmt.Printf("👥 Pejabat/Admin : %d akun\n", summary.Officials)
	fmt.Printf("🚩 Kontingen     : %d kontingen\n", summary.Contingents)
	fmt.Printf("🥋 Atlet         : %d atlet\n", summary.Athletes)
	fmt.Printf("📋 Peserta Bagan : %d peserta (Kelas A Putra)\n", summary.Participants)
	fmt.Printf("⚔️ Pertandingan  : %d match aktif siap tanding\n", summary.Matches)
	fmt.Printf("🔑 PIN Juri/Wasit: %d PIN terkonfigurasi\n", summary.JudgePins)

	fmt.Println("\n================================================================================")
	fmt.Println("🔐 KREDENSIAL AKSES SISTEM (LOGIN ACCOUNTS & PINS)")
	fmt.Println("================================================================================")

	fmt.Println("\n1. OFFICIAL & ADMIN (POST /api/v1/auth/login)")
	fmt.Println("   Role       | Email              | Password   | Akses & Kegunaan")
	fmt.Println("   -----------+--------------------+------------+-----------------------------------")
	fmt.Println("   admin      | admin@silat.id     | DemoPass_2026  | Full Dashboard, Buat PIN & Bagan")
	fmt.Println("   dewan      | dewan@silat.id     | DemoPass_2026  | Evaluasi Pertandingan & Pengawasan")
	fmt.Println("   wasit      | wasit@silat.id     | DemoPass_2026  | Wasit Resmi Pertandingan")
	fmt.Println("   operator   | operator@silat.id  | DemoPass_2026| Kendali Layar & Operator Gelanggang")

	fmt.Println("\n2. KONTINGEN / MANAJER TIM (POST /api/v1/auth/contingent/login)")
	fmt.Println("   Kontingen                        | Email                     | Password")
	fmt.Println("   ---------------------------------+---------------------------+--------------")
	fmt.Println("   Tapak Suci Putera Muhammadiyah   | tapaksuci@contingent.id   | DemoPass_2026")
	fmt.Println("   PSHT Jawa Timur                  | psht@contingent.id        | DemoPass_2026")
	fmt.Println("   PPS Betako Merpati Putih         | merpatiputih@contingent.id| DemoPass_2026")
	fmt.Println("   KSN Perisai Diri                 | perisaidiri@contingent.id | DemoPass_2026")

	fmt.Println("\n3. WASIT & JURI GELANGGANG A (POST /api/v1/auth/judge-login)")
	fmt.Printf("   Arena ID: %s\n", summary.ArenaAID)
	fmt.Printf("   Competition ID: %s\n", summary.CompetitionID)
	fmt.Println("   Role       | Kursi / Seat | PIN     | Antarmuka")
	fmt.Println("   -----------+--------------+---------+------------------------------")
	fmt.Println("   wasit      | -            | 11110001| Referee Control Pad (Timer & Sanksi)")
	fmt.Println("   juri       | Seat 1       | 123001  | Judge 1 Scoring Tablet PWA")
	fmt.Println("   juri       | Seat 2       | 123002  | Judge 2 Scoring Tablet PWA")
	fmt.Println("   juri       | Seat 3       | 123003  | Judge 3 Scoring Tablet PWA")

	fmt.Println("\n4. WASIT & JURI GELANGGANG B (POST /api/v1/auth/judge-login)")
	fmt.Printf("   Arena ID: %s\n", summary.ArenaBID)
	fmt.Printf("   Competition ID: %s\n", summary.CompetitionID)
	fmt.Println("   Role       | Kursi / Seat | PIN     | Antarmuka")
	fmt.Println("   -----------+--------------+---------+------------------------------")
	fmt.Println("   wasit      | -            | 22220001| Referee Control Pad (Timer & Sanksi)")
	fmt.Println("   juri       | Seat 1       | 234001  | Judge 1 Scoring Tablet PWA")
	fmt.Println("   juri       | Seat 2       | 234002  | Judge 2 Scoring Tablet PWA")
	fmt.Println("   juri       | Seat 3       | 234003  | Judge 3 Scoring Tablet PWA")

	fmt.Println("\n5. SCOREBOARD & PUBLIC HUB")
	fmt.Printf("   Live Score Gelanggang A : /api/v1/live/scoreboard/%s\n", summary.ArenaAID)
	fmt.Printf("   Live Score Gelanggang B : /api/v1/live/scoreboard/%s\n", summary.ArenaBID)
	fmt.Printf("   Bagan Turnamen (Tree)   : /api/v1/competitions/%s/brackets\n", summary.CompetitionID)
	fmt.Println("================================================================================")
	os.Exit(0)
}
