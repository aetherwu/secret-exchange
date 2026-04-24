import NavBar from "@/components/NavBar";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="pb-16 md:pb-0 md:pt-14">
      <NavBar />
      <main>{children}</main>
    </div>
  );
}
