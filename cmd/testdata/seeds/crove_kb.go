package seeds

type CroveFAQItem struct {
	Question         string
	Answer           string
	SimilarQuestions []string
	Remark           string
}

func CroveDeskFAQSeeds() []CroveFAQItem {
	return []CroveFAQItem{
		{
			Question: "Crove Desk là gì và đóng vai trò gì trong hệ sinh thái Crove Business OS?",
			Answer:   "Crove Desk là nền tảng AI Customer Support & HelpDesk thông minh thuộc hệ sinh thái Crove Business OS (cùng với Crove CRM, Crove Sign, Crove Post và Crove Cal). Crove Desk đóng vai trò là bộ não giao tiếp khách hàng (AI Communication Engine), tự động hóa hỗ trợ 24/7 bằng AI RAG, tiếp nhận tin nhắn đa kênh, quản lý Ticket và kết nối dữ liệu khách hàng 2 chiều với Crove CRM qua giao thức MCP.",
			SimilarQuestions: []string{
				"Giới thiệu về Crove Desk",
				"Crove Desk dùng để làm gì",
				"Crove Desk khác gì các phần mềm CSKH khác",
			},
			Remark: "Tổng quan sản phẩm",
		},
		{
			Question: "Cơ chế Answerability Gate trong Crove Desk hoạt động như thế nào?",
			Answer:   "Answerability Gate là cơ chế kiểm soát chất lượng câu trả lời của AI. Khi khách hàng đặt câu hỏi, hệ thống thực hiện truy xuất tài liệu từ Vector DB (Qdrant) và Answerability Gate sẽ đánh giá xem các đoạn trích dẫn có đủ bằng chứng và độ tin cậy để trả lời hay không. Nếu dữ liệu không đủ hoặc vượt ngoài phạm vi kiến thức, AI sẽ kích hoạt chính sách Fallback (trả lời an toàn, tránh bịa đặt thông tin) và đề xuất chuyển giao cuộc hội thoại cho nhân viên hỗ trợ (Human Agent Handoff).",
			SimilarQuestions: []string{
				"Answerability Gate là gì",
				"Làm sao AI không bịa câu trả lời",
				"Cơ chế chống ảo giác hallucination trong Crove Desk",
			},
			Remark: "AI & RAG Engine",
		},
		{
			Question: "Crove Desk tích hợp với Crove CRM (Twenty CRM) qua giao thức MCP như thế nào?",
			Answer:   "Crove Desk và Crove CRM tích hợp sâu qua Model Context Protocol (MCP) 2 chiều: 1) AI của Crove Desk có thể gọi MCP Tools của Twenty CRM để tra cứu Company, Contact, Deal, gói đăng ký của khách hàng theo thời gian thực; 2) Khi khách chat có nhu cầu mua hàng, AI tự động tạo Opportunity và Task mới trên CRM; 3) Ngược lại, đội ngũ bán hàng trên Twenty CRM có thể tra cứu lịch sử chat và ticket của khách hàng thông qua CRM UI Widget.",
			SimilarQuestions: []string{
				"Tích hợp Crove Desk với Twenty CRM",
				"MCP Protocol trong Crove Desk hoạt động ra sao",
				"Đồng bộ dữ liệu khách hàng với CRM",
			},
			Remark: "Tích hợp MCP & CRM",
		},
		{
			Question: "Làm thế nào để nhúng Web Chat Widget của Crove Desk vào website?",
			Answer:   "Để nhúng Web Chat Widget, bạn vào Quản trị > Kênh liên lạc (Channels) > Web Widget, cấu hình thương hiệu (Tên, Logo, Màu chủ đạo, Lời chào) và sao chép đoạn mã JavaScript SDK (`agent-desk-sdk.min.js`). Dán đoạn mã này vào trước thẻ `</body>` trên trang web của bạn. Widget hỗ trợ cả giao diện Desktop và Mobile mượt mà, đồng thời tự động đồng bộ danh tính người dùng khi đã đăng nhập.",
			SimilarQuestions: []string{
				"Cách cài đặt widget chat lên web",
				"Nhúng nút chat hỗ trợ vào website",
				"Cài đặt SDK Crove Desk",
			},
			Remark: "Kênh liên lạc",
		},
		{
			Question: "Quy trình chuyển đổi cuộc hội thoại thành Ticket (Ticket Lifecycle) diễn ra như thế nào?",
			Answer:   "Khi khách hàng gặp vấn đề phức tạp cần theo dõi, Agent hoặc AI có thể chuyển đổi cuộc hội thoại thành Ticket. Ticket sẽ được gán mã định danh duy nhất (VD: TK-2026-0001), phân loại danh mục, độ ưu tiên (Low, Medium, High, Urgent), người phụ trách (Assignee) và hạn xử lý SLA. Toàn bộ lịch sử trao đổi, ghi chú nội bộ và tiến độ xử lý được lưu vết tập trung.",
			SimilarQuestions: []string{
				"Cách tạo ticket hỗ trợ",
				"Quản lý vòng đời ticket",
				"Phân công xử lý ticket cho nhân viên",
			},
			Remark: "Quản lý Ticket",
		},
		{
			Question: "Crove Desk hỗ trợ các mô hình AI (LLM & Embedding) nào?",
			Answer:   "Crove Desk hỗ trợ tất cả các nhà cung cấp AI tương thích chuẩn OpenAI API (OpenAI-compatible), bao gồm DOS.AI (model `dos-ai`, embedding `qwen3-embedding-4b`), OpenAI (GPT-4o, GPT-4o-mini, text-embedding-3-small), DeepSeek, OpenRouter, Azure OpenAI, vLLM và Ollama chạy local. Cấu hình có thể nạp tự động qua biến môi trường (.env) hoặc quản lý trực tiếp trong trang quản trị AI Configs.",
			SimilarQuestions: []string{
				"Cấu hình LLM cho Crove Desk",
				"Crove Desk dùng model AI nào",
				"Hỗ trợ DeepSeek và OpenAI không",
			},
			Remark: "Cấu hình AI",
		},
		{
			Question: "Cơ chế Single Sign-On (SSO) và Multi-tenancy trong Crove Desk hoạt động ra sao?",
			Answer:   "Crove Desk hỗ trợ đăng nhập một chạm (SSO) qua giao thức OIDC / OAuth 2.1 với chuẩn bảo mật PKCE S256 (kết nối trực tiếp với DOS ID / Supabase Auth). Đồng thời, hệ thống hỗ trợ đa tổ chức (Multi-tenancy/Workspaces) cho phép người dùng chuyển đổi linh hoạt giữa các Workspace khác nhau với cơ chế đồng bộ 2 pha Hybrid Sync (JIT Provisioning khi đăng nhập và Realtime Webhook Sync).",
			SimilarQuestions: []string{
				"Đăng nhập bằng DOS ID",
				"Multi-tenant trong Crove Desk",
				"Chuyển đổi workspace tổ chức",
			},
			Remark: "Bảo mật & Xác thực",
		},
	}
}
